package hr_service

import (
	"log"
	"strings"
	"time"

	"github.com/zapscloud/golib-business-repository/business_common"
	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_common"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
)

// LeaveService - Accounts Service structure
type LeaveService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(leaveId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(leaveId string, indata utils.Map) (utils.Map, error)
	Delete(leaveId string, delete_permanent bool) error
	DeleteAll(delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// leaveBaseService - Accounts Service structure
type leaveBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoLeave            hr_repository.LeaveDao
	daoPlatformBusiness platform_repository.BusinessDao
	daoPlatformAppUser  platform_repository.AppUserDao
	daoStaff            hr_repository.StaffDao

	child      LeaveService
	businessId string
	staffId    string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewLeaveService(props utils.Map) (LeaveService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("LeaveService::Start")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := leaveBaseService{}

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

	// Verify whether the User id data passed, this is optional parameter
	staffId, _ := utils.GetMemberDataStr(props, hr_common.FLD_STAFF_ID)

	// Assign the BusinessId & StaffId
	p.businessId = businessId
	p.staffId = staffId

	// Instantiate other services
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())
	p.daoPlatformAppUser = platform_repository.NewAppUserDao(p.GetClient())
	p.daoLeave = hr_repository.NewLeaveDao(p.dbRegion.GetClient(), p.businessId, p.staffId)
	p.daoStaff = hr_repository.NewStaffDao(p.dbRegion.GetClient(), p.businessId)

	_, err = p.daoPlatformBusiness.Get(p.businessId)
	if err != nil {
		err := &utils.AppError{
			ErrorCode:   funcode + "01",
			ErrorMsg:    "Invalid business id",
			ErrorDetail: "Given business id is not exist"}
		return p.errorReturn(err)
	}

	// Verify the Staff Exist
	if len(staffId) > 0 {
		_, err = p.daoStaff.Get(staffId)
		if err != nil {
			err := &utils.AppError{
				ErrorCode:   funcode + "01",
				ErrorMsg:    "Invalid StaffId",
				ErrorDetail: "Given StaffId is not exist"}
			return p.errorReturn(err)
		}
	}

	p.child = &p

	return &p, nil
}

func (p *leaveBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *leaveBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoLeave := p.daoLeave
	response, err := daoLeave.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	// Lookup Appuser Info
	//p.lookupAppuser(response)

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *leaveBaseService) Get(leaveId string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", leaveId)

	data, err := p.daoLeave.Get(leaveId)
	log.Println("AccountService::FindByCode:: End ", err)
	// Lookup Appuser Info
	p.mergeUserInfo(data)
	return data, err
}

func (p *leaveBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoLeave.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *leaveBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var leaveId string

	dataval, dataok := indata[hr_common.FLD_LEAVE_ID]
	if dataok {
		leaveId = strings.ToLower(dataval.(string))
	} else {
		leaveId = utils.GenerateUniqueId("leav")
		log.Println("Unique Account ID", leaveId)
	}
	indata[hr_common.FLD_LEAVE_ID] = leaveId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessId
	indata[hr_common.FLD_STAFF_ID] = p.staffId
	log.Println("Provided Account ID:", leaveId)

	_, err := p.daoLeave.Get(leaveId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Account ID !", ErrorDetail: "Given Account ID already exist"}
		return utils.Map{}, err
	}
	err = p.validateDateTime(indata)
	if err != nil {
		return utils.Map{}, err
	}

	insertResult, err := p.daoLeave.Create(indata)
	if err != nil {
		return utils.Map{}, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *leaveBaseService) Update(leaveId string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoLeave.Get(leaveId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_LEAVE_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)
	delete(indata, hr_common.FLD_STAFF_ID)

	err = p.validateDateTime(indata)
	if err != nil {
		return utils.Map{}, err
	}

	data, err = p.daoLeave.Update(leaveId, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *leaveBaseService) Delete(leaveId string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", leaveId)

	daoLeave := p.daoLeave
	_, err := daoLeave.Get(leaveId)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoLeave.Delete(leaveId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoLeave.Update(leaveId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("LeaveService::Delete - End")
	return nil
}

// ***********************************************
// DeleteAll - Delete All Leaves/Permissions for the staff
//
// ***********************************************
func (p *leaveBaseService) DeleteAll(delete_permanent bool) error {

	log.Println("LeaveService::DeleteAll - Begin", delete_permanent)

	daoLeave := p.daoLeave
	if delete_permanent {
		result, err := daoLeave.DeleteMany()
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoLeave.UpdateMany(indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("LeaveService::DeleteAll - End")
	return nil
}

func (p *leaveBaseService) errorReturn(err error) (LeaveService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}

func (p *leaveBaseService) validateDateTime(indata utils.Map) error {

	// Convert Leave_From string to Date Format
	dateTime, err := utils.GetMemberDataStr(indata, hr_common.FLD_LEAVE_FROM)
	if err == nil {
		_, err = time.Parse(time.DateTime, dateTime)
		if err != nil {
			err = &utils.AppError{
				ErrorCode:   "S30102",
				ErrorMsg:    "Invalid leave_from",
				ErrorDetail: "leave_from value is invalid"}
			return err
		}
	}

	// Convert Leave_To string to Date Format
	dateTime, err = utils.GetMemberDataStr(indata, hr_common.FLD_LEAVE_TO)
	if err == nil {
		_, err = time.Parse(time.DateTime, dateTime)
		if err != nil {
			err = &utils.AppError{
				ErrorCode:   "S30102",
				ErrorMsg:    "Invalid leave_to",
				ErrorDetail: "leave_to value is invalid"}
			return err
		}
	}
	return nil
}

func (p *leaveBaseService) lookupAppuser(response utils.Map) {

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

func (p *leaveBaseService) mergeUserInfo(staffInfo utils.Map) {

	staffId, _ := utils.GetMemberDataStr(staffInfo, hr_common.FLD_STAFF_ID)
	staffData, err := p.daoPlatformAppUser.Get(staffId)
	if err == nil {
		// Delete unwanted fields
		delete(staffData, db_common.FLD_CREATED_AT)
		delete(staffData, db_common.FLD_UPDATED_AT)
		delete(staffData, platform_common.FLD_APP_USER_ID)

		// Make it as Array for backward compatible, since all MongoDB Lookups data returned as array
		staffInfo[business_common.FLD_USER_INFO] = []utils.Map{staffData}
	}
}
