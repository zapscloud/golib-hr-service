package hr_service

import (
	"log"
	"strings"

	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
)

// DepartmentService - Departments Service structure
type DepartmentService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(departmentid string) (utils.Map, error)
	GetDeptCodeDetails(departmentcode string) (utils.Map, error)

	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(departmentid string, indata utils.Map) (utils.Map, error)
	Delete(departmentid string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

type departmentBaseService struct {
	db_utils.DatabaseService
	dbRegion      db_utils.DatabaseService
	daoDepartment hr_repository.DepartmentDao
	daoBusiness   platform_repository.BusinessDao
	child         DepartmentService
	businessID    string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewDepartmentService(props utils.Map) (DepartmentService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("DepartmentService::Start ")
	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := departmentBaseService{}
	err = p.OpenDatabaseService(props)
	if err != nil {
		return nil, err
	}

	// Open RegionDB Service
	p.dbRegion, err = platform_service.OpenRegionDatabaseService(props)
	if err != nil {
		p.CloseDatabaseService()
		return nil, err
	}

	// Assign the BusinessId
	p.businessID = businessId
	p.initializeService()

	_, err = p.daoBusiness.Get(businessId)
	if err != nil {
		err := &utils.AppError{
			ErrorCode:   funcode + "01",
			ErrorMsg:    "Invalid business_id",
			ErrorDetail: "Given app_business_id is not exist"}
		return p.errorReturn(err)
	}

	p.child = &p

	return &p, err
}

func (p *departmentBaseService) EndService() {
	log.Printf("EndDepartmentMongoService ")
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

func (p *departmentBaseService) initializeService() {
	log.Printf("DepartmentMongoService:: GetBusinessDao ")
	p.daoDepartment = hr_repository.NewDepartmentDao(p.dbRegion.GetClient(), p.businessID)
	p.daoBusiness = platform_repository.NewBusinessDao(p.GetClient())
}

// List - List All records
func (p *departmentBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("DepartmentService::FindAll - Begin")

	daoDepartment := p.daoDepartment
	response, err := daoDepartment.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("DepartmentService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *departmentBaseService) GetDeptCodeDetails(department_code string) (utils.Map, error) {
	log.Printf("DepartmentService::FindByCode::  Begin %v", department_code)

	data, err := p.daoDepartment.GetDeptCodeDetails(department_code)
	log.Println("DepartmentService::FindByCode:: End ", err)
	return data, err
}

// FindByCode - Find By Code
func (p *departmentBaseService) Get(department_id string) (utils.Map, error) {
	log.Printf("DepartmentService::FindByCode::  Begin %v", department_id)

	data, err := p.daoDepartment.Get(department_id)
	log.Println("DepartmentService::FindByCode:: End ", err)
	return data, err
}

func (p *departmentBaseService) Find(filter string) (utils.Map, error) {
	log.Println("DepartmentService::FindByCode::  Begin ", filter)

	data, err := p.daoDepartment.Find(filter)
	log.Println("DepartmentService::FindByCode:: End ", data, err)
	return data, err
}

func (p *departmentBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")
	var deptId string

	dataval, dataok := indata[hr_common.FLD_DEPARTMENT_ID]
	if dataok {
		deptId = strings.ToLower(dataval.(string))
	} else {
		deptId = utils.GenerateUniqueId("dept")
		log.Println("Unique Department ID", deptId)
	}

	indata[hr_common.FLD_DEPARTMENT_ID] = deptId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Department ID:", dataval)

	_, err := p.daoDepartment.Get(deptId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Department ID !", ErrorDetail: "Given Department ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoDepartment.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *departmentBaseService) Update(department_id string, indata utils.Map) (utils.Map, error) {

	log.Println("DepartmentService::Update - Begin")

	data, err := p.daoDepartment.Get(department_id)
	if err != nil {
		return data, err
	}
	// Delete unique fields
	delete(indata, hr_common.FLD_BUSINESS_ID)
	delete(indata, hr_common.FLD_DEPARTMENT_ID)

	data, err = p.daoDepartment.Update(department_id, indata)
	log.Println("DepartmentService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *departmentBaseService) Delete(department_id string, delete_permanent bool) error {

	log.Println("DepartmentService::Delete - Begin", department_id, delete_permanent)

	daoDepartment := p.daoDepartment
	_, err := daoDepartment.Get(department_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoDepartment.Delete(department_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoDepartment.Update(department_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("DepartmentService::Delete - End")
	return nil
}

func (p *departmentBaseService) errorReturn(err error) (DepartmentService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
