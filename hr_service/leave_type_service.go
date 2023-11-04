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

// LeaveTypeService - LeaveTypes Service structure
type LeaveTypeService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(LeaveTypeid string) (utils.Map, error)
	GetDeptCodeDetails(LeaveTypecode string) (utils.Map, error)

	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(LeaveTypeid string, indata utils.Map) (utils.Map, error)
	Delete(LeaveTypeid string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

type leaveTypeBaseService struct {
	db_utils.DatabaseService
	dbRegion     db_utils.DatabaseService
	daoLeaveType hr_repository.LeaveTypeDao
	daoBusiness  platform_repository.BusinessDao
	child        LeaveTypeService
	businessID   string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewLeaveTypeService(props utils.Map) (LeaveTypeService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("LeaveTypeService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := leaveTypeBaseService{}
	// Open Database Service
	err = p.OpenDatabaseService(props)
	if err != nil {
		return nil, err
	}

	// Open RegionDB Service
	p.dbRegion, err = platform_services.OpenRegionDatabaseService(props)
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

func (p *leaveTypeBaseService) EndService() {
	log.Printf("EndLeaveTypeMongoService ")
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

func (p *leaveTypeBaseService) initializeService() {
	log.Printf("LeaveTypeMongoService:: GetBusinessDao ")
	p.daoLeaveType = hr_repository.NewLeaveTypeDao(p.dbRegion.GetClient(), p.businessID)
	p.daoBusiness = platform_repository.NewBusinessDao(p.GetClient())
}

// List - List All records
func (p *leaveTypeBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("LeaveTypeService::FindAll - Begin")

	daoLeaveType := p.daoLeaveType
	response, err := daoLeaveType.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("LeaveTypeService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *leaveTypeBaseService) GetDeptCodeDetails(LeaveType_code string) (utils.Map, error) {
	log.Printf("LeaveTypeService::FindByCode::  Begin %v", LeaveType_code)

	data, err := p.daoLeaveType.GetDeptCodeDetails(LeaveType_code)
	log.Println("LeaveTypeService::FindByCode:: End ", err)
	return data, err
}

// FindByCode - Find By Code
func (p *leaveTypeBaseService) Get(LeaveType_id string) (utils.Map, error) {
	log.Printf("LeaveTypeService::FindByCode::  Begin %v", LeaveType_id)

	data, err := p.daoLeaveType.Get(LeaveType_id)
	log.Println("LeaveTypeService::FindByCode:: End ", err)
	return data, err
}

func (p *leaveTypeBaseService) Find(filter string) (utils.Map, error) {
	log.Println("LeaveTypeService::FindByCode::  Begin ", filter)

	data, err := p.daoLeaveType.Find(filter)
	log.Println("LeaveTypeService::FindByCode:: End ", data, err)
	return data, err
}

func (p *leaveTypeBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")
	var deptId string

	dataval, dataok := indata[hr_common.FLD_LEAVETYPE_ID]
	if dataok {
		deptId = strings.ToLower(dataval.(string))
	} else {
		deptId = utils.GenerateUniqueId("ltype")
		log.Println("Unique LeaveType ID", deptId)
	}

	indata[hr_common.FLD_LEAVETYPE_ID] = deptId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided LeaveType ID:", dataval)

	_, err := p.daoLeaveType.Get(deptId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing LeaveType ID !", ErrorDetail: "Given LeaveType ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoLeaveType.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *leaveTypeBaseService) Update(LeaveType_id string, indata utils.Map) (utils.Map, error) {

	log.Println("LeaveTypeService::Update - Begin")

	data, err := p.daoLeaveType.Get(LeaveType_id)
	if err != nil {
		return data, err
	}
	// Delete unique fields
	delete(indata, hr_common.FLD_BUSINESS_ID)
	delete(indata, hr_common.FLD_LEAVETYPE_ID)

	data, err = p.daoLeaveType.Update(LeaveType_id, indata)
	log.Println("LeaveTypeService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *leaveTypeBaseService) Delete(LeaveType_id string, delete_permanent bool) error {

	log.Println("LeaveTypeService::Delete - Begin", LeaveType_id, delete_permanent)

	daoLeaveType := p.daoLeaveType
	_, err := daoLeaveType.Get(LeaveType_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoLeaveType.Delete(LeaveType_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoLeaveType.Update(LeaveType_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("LeaveTypeService::Delete - End")
	return nil
}

func (p *leaveTypeBaseService) errorReturn(err error) (LeaveTypeService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
