package hr_service

import (
	"log"

	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
)

// StaffTypeService - Accounts Service structure
type StaffTypeService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(staffTypeId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(staffTypeId string, indata utils.Map) (utils.Map, error)
	Delete(staffTypeId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// staffTypeBaseService - Accounts Service structure
type staffTypeBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoStaffType        hr_repository.StaffTypeDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               StaffTypeService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewStaffTypeService(props utils.Map) (StaffTypeService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("StaffTypeService::Start ")
	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := staffTypeBaseService{}

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

	// Instantiate other services
	p.daoStaffType = hr_repository.NewStaffTypeDao(p.dbRegion.GetClient(), p.businessID)
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())

	_, err = p.daoPlatformBusiness.Get(p.businessID)
	if err != nil {
		err := &utils.AppError{
			ErrorCode:   funcode + "01",
			ErrorMsg:    "Invalid business id",
			ErrorDetail: "Given business id is not exist"}
		return p.errorReturn(err)
	}

	p.child = &p

	return &p, nil
}

func (p *staffTypeBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *staffTypeBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoStaffType := p.daoStaffType
	response, err := daoStaffType.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *staffTypeBaseService) Get(staffTypeId string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", staffTypeId)

	data, err := p.daoStaffType.Get(staffTypeId)
	log.Println("AccountService::FindByCode:: End ", err)
	return data, err
}

func (p *staffTypeBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoStaffType.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *staffTypeBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	dataval, dataok := indata[hr_common.FLD_STAFFTYPE_ID]
	if !dataok {
		uid := utils.GenerateUniqueId("stftyp")
		log.Println("Unique Account ID", uid)
		indata[hr_common.FLD_STAFFTYPE_ID] = uid
		dataval = indata[hr_common.FLD_STAFFTYPE_ID]
	}
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Account ID:", dataval)

	_, err := p.daoStaffType.Get(dataval.(string))
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Account ID !", ErrorDetail: "Given Account ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoStaffType.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *staffTypeBaseService) Update(staffTypeId string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoStaffType.Get(staffTypeId)
	if err != nil {
		return data, err
	}

	data, err = p.daoStaffType.Update(staffTypeId, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *staffTypeBaseService) Delete(staffTypeId string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", staffTypeId)

	daoStaffType := p.daoStaffType
	_, err := daoStaffType.Get(staffTypeId)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoStaffType.Delete(staffTypeId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoStaffType.Update(staffTypeId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("StaffTypeService::Delete - End")
	return nil
}

func (p *staffTypeBaseService) errorReturn(err error) (StaffTypeService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
