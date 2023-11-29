package hr_service

import (
	"log"

	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_common"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
)

// StaffService - Accounts Service structure
type StaffService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(staff_id string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(staff_id string, indata utils.Map) (utils.Map, error)
	Delete(staff_id string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// staffBaseService - Accounts Service structure
type staffBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoStaff            hr_repository.StaffDao
	daoPlatformBusiness platform_repository.BusinessDao
	daoPlatformAppUser  platform_repository.AppUserDao
	child               StaffService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewStaffService(props utils.Map) (StaffService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("StaffService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := staffBaseService{}

	// Open Database Service
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

	// Instantiate other services
	p.daoStaff = hr_repository.NewStaffDao(p.dbRegion.GetClient(), p.businessID)
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())
	p.daoPlatformAppUser = platform_repository.NewAppUserDao(p.GetClient())

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

func (p *staffBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *staffBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoStaff := p.daoStaff
	response, err := daoStaff.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	// Lookup Appuser Info
	p.lookupAppuser(response)
	p.lookupreportingstaff(response)

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *staffBaseService) Get(staff_id string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", staff_id)

	data, err := p.daoStaff.Get(staff_id)
	log.Println("AccountService::FindByCode:: End ", err)
	return data, err
}

func (p *staffBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoStaff.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *staffBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	dataval, dataok := indata[hr_common.FLD_STAFF_ID]
	if !dataok {
		uid := utils.GenerateUniqueId("stf")
		log.Println("Unique Account ID", uid)
		indata[hr_common.FLD_STAFF_ID] = uid
		dataval = indata[hr_common.FLD_STAFF_ID]
	}
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Account ID:", dataval)

	_, err := p.daoStaff.Get(dataval.(string))
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Account ID !", ErrorDetail: "Given Account ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoStaff.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *staffBaseService) Update(staff_id string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoStaff.Get(staff_id)
	if err != nil {
		return data, err
	}

	data, err = p.daoStaff.Update(staff_id, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *staffBaseService) Delete(staff_id string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", staff_id)

	daoStaff := p.daoStaff
	_, err := daoStaff.Get(staff_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoStaff.Delete(staff_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoStaff.Update(staff_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("StaffService::Delete - End")
	return nil
}

func (p *staffBaseService) errorReturn(err error) (StaffService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}

func (p *staffBaseService) lookupAppuser(response utils.Map) {

	// Enumerate All staffs and lookup platform_app_user table
	dataStaff, err := utils.GetMemberData(response, db_common.LIST_RESULT)

	if err == nil {
		staffs := dataStaff.([]utils.Map)
		for _, staff := range staffs {
			p.mergeUserInfo(staff)
			//log.Println(staff)
		}
	}
}

func (p *staffBaseService) mergeUserInfo(staffInfo utils.Map) {

	staffId, _ := utils.GetMemberDataStr(staffInfo, hr_common.FLD_STAFF_ID)
	staffData, err := p.daoPlatformAppUser.Get(staffId)
	if err == nil {
		// Delete unwanted fields
		delete(staffData, db_common.FLD_CREATED_AT)
		delete(staffData, db_common.FLD_UPDATED_AT)
		delete(staffData, platform_common.FLD_APP_USER_ID)

		// Make it as Array for backward compatible, since all MongoDB Lookups data returned as array
		staffInfo[hr_common.FLD_STAFF_INFO] = []utils.Map{staffData}
	}
}

func (p *staffBaseService) lookupreportingstaff(response utils.Map) {

	// Enumerate All staffs and lookup platform_app_user table
	dataStaff, err := utils.GetMemberData(response, db_common.LIST_RESULT)
	if err == nil {
		staffs := dataStaff.([]utils.Map)
		for _, staff := range staffs {
			p.mergereportingInfo(staff)
		}
	}
}

func (p *staffBaseService) mergereportingInfo(reportingstaffInfo utils.Map) {
	staffDataInterface, _ := reportingstaffInfo[hr_common.FLD_STAFF_DATA]

	staffDatas, _ := staffDataInterface.(utils.Map)

	reportingstaffId, err := utils.GetMemberDataStr(staffDatas, hr_common.FLD_REPORTING_STAFF_ID)

	staffData, err := p.daoPlatformAppUser.Get(reportingstaffId)
	if err == nil {
		// Delete unwanted fields
		delete(staffData, db_common.FLD_CREATED_AT)
		delete(staffData, db_common.FLD_UPDATED_AT)
		delete(staffData, platform_common.FLD_APP_USER_ID)

		// Make it as Array for backward compatible, since all MongoDB Lookups data returned as array
		reportingstaffInfo[hr_common.FLD_REPORTING_STAFF_INFO] = []utils.Map{staffData}
	}
}
