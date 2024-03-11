package hr_service

import (
	"log"
	"strings"
	"time"

	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
)

// Staff_categoryService - Accounts Service structure
type Staff_categoryService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(Staff_categoryId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(Staff_categoryId string, indata utils.Map) (utils.Map, error)
	Delete(Staff_categoryId string, delete_permanent bool) error
	DeleteAll(delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// Staff_categoryBaseService - Accounts Service structure
type Staff_categoryBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoLeave            hr_repository.LeaveDao
	daoPlatformBusiness platform_repository.BusinessDao
	daoPlatformAppUser  platform_repository.AppUserDao
	daoStaff            hr_repository.StaffDao

	child      Staff_categoryService
	businessId string
	staffId    string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewStaff_categoryService(props utils.Map) (Staff_categoryService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("Staff_categoryService::Start")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := Staff_categoryBaseService{}

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

func (p *Staff_categoryBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *Staff_categoryBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoLeave := p.daoLeave
	response, err := daoLeave.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	// // Lookup Appuser Info
	// p.lookupAppuser(response)

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *Staff_categoryBaseService) Get(Staff_categoryId string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", Staff_categoryId)

	data, err := p.daoLeave.Get(Staff_categoryId)
	log.Println("AccountService::FindByCode:: End ", err)
	return data, err
}

func (p *Staff_categoryBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoLeave.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *Staff_categoryBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var Staff_categoryId string

	dataval, dataok := indata[hr_common.FLD_STAFF_CATEGORY_ID]
	if dataok {
		Staff_categoryId = strings.ToLower(dataval.(string))
	} else {
		Staff_categoryId = utils.GenerateUniqueId("stfcat")
		log.Println("Unique Account ID", Staff_categoryId)
	}
	indata[hr_common.FLD_STAFF_CATEGORY_ID] = Staff_categoryId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessId
	log.Println("Provided Account ID:", Staff_categoryId)

	_, err := p.daoLeave.Get(Staff_categoryId)
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
func (p *Staff_categoryBaseService) Update(Staff_categoryId string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoLeave.Get(Staff_categoryId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_STAFF_CATEGORY_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)
	delete(indata, hr_common.FLD_STAFF_ID)

	err = p.validateDateTime(indata)
	if err != nil {
		return utils.Map{}, err
	}

	data, err = p.daoLeave.Update(Staff_categoryId, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *Staff_categoryBaseService) Delete(Staff_categoryId string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", Staff_categoryId)

	daoLeave := p.daoLeave
	_, err := daoLeave.Get(Staff_categoryId)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoLeave.Delete(Staff_categoryId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoLeave.Update(Staff_categoryId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("Staff_categoryService::Delete - End")
	return nil
}

// ***********************************************
// DeleteAll - Delete All Leaves/Permissions for the staff
//
// ***********************************************
func (p *Staff_categoryBaseService) DeleteAll(delete_permanent bool) error {

	log.Println("Staff_categoryService::DeleteAll - Begin", delete_permanent)

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

	log.Printf("Staff_categoryService::DeleteAll - End")
	return nil
}

func (p *Staff_categoryBaseService) errorReturn(err error) (Staff_categoryService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}

func (p *Staff_categoryBaseService) validateDateTime(indata utils.Map) error {

	// Convert Leave_From string to Date Format
	dateTime, err := utils.GetMemberDataStr(indata, hr_common.FLD_LEAVE_FROM)
	if err == nil {
		_, err = time.Parse(time.DateTime, dateTime)
		if err != nil {
			err = &utils.AppError{
				ErrorCode:   "S30102",
				ErrorMsg:    "Invalid Staff_category_from",
				ErrorDetail: "Staff_category_from value is invalid"}
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
				ErrorMsg:    "Invalid Staff_category_to",
				ErrorDetail: "Staff_category_to value is invalid"}
			return err
		}
	}
	return nil
}

// func (p *Staff_categoryBaseService) lookupAppuser(response utils.Map) {

// 	// Enumerate All staffs and lookup platform_app_user table
// 	dataStaff, err := utils.GetMemberData(response, db_common.LIST_RESULT)

// 	if err == nil {
// 		staffs := dataStaff.([]utils.Map)
// 		for _, staff := range staffs {
// 			p.mergeUserInfo(staff)
// 			//log.Println(staff)
// 		}
// 	}
// }

// func (p *Staff_categoryBaseService) mergeUserInfo(staffInfo utils.Map) {

// 	staffId, _ := utils.GetMemberDataStr(staffInfo, hr_common.FLD_STAFF_ID)
// 	staffData, err := p.daoPlatformAppUser.Get(staffId)
// 	if err == nil {
// 		// Delete unwanted fields
// 		delete(staffData, db_common.FLD_CREATED_AT)
// 		delete(staffData, db_common.FLD_UPDATED_AT)
// 		delete(staffData, platform_common.FLD_APP_USER_ID)

// 		// Make it as Array for backward compatible, since all MongoDB Lookups data returned as array
// 		staffInfo[hr_common.FLD_STAFF_INFO] = []utils.Map{staffData}
// 	}
// }
