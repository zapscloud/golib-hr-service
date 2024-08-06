package hr_service

import (
	"log"
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

const (
// // The character encoding for the email.
// CharSet = "UTF-8"
)

// AttendanceService - Attendances Service structure
type AttendanceService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	ListNew(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(attendance_id string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	// Create(indata utils.Map) (utils.Map, error)
	// CreateMany(indata utils.Map) (utils.Map, error)
	ClockIn(indata utils.Map) (utils.Map, error)
	ClockInMany(indata utils.Map) (utils.Map, error)
	ClockOut(attendance_id string, indata utils.Map) (utils.Map, error)
	ClockOutMany(indata utils.Map) (utils.Map, error)
	Update(attendance_id string, indata utils.Map) (utils.Map, error)
	Delete(attendance_id string, delete_permanent bool) error
	DeleteAll(delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// AttendanceBaseService - Attendances Service structure
type attendanceBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoAttendance       hr_repository.AttendanceDao
	daoPlatformBusiness platform_repository.BusinessDao
	daoPlatformAppUser  platform_repository.AppUserDao
	daoStaff            hr_repository.StaffDao

	child      AttendanceService
	businessId string
	staffId    string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewAttendanceService(props utils.Map) (AttendanceService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("AttendanceSerivce::Start ")
	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := attendanceBaseService{}
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

	// Initialize services
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())
	p.daoPlatformAppUser = platform_repository.NewAppUserDao(p.GetClient())
	p.daoAttendance = hr_repository.NewAttendanceDao(p.dbRegion.GetClient(), p.businessId, p.staffId)
	p.daoStaff = hr_repository.NewStaffDao(p.dbRegion.GetClient(), p.businessId)

	// Verify the BusinessId is exist
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

func (p *attendanceBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// ************************
// List - List All records
//
// ************************
func (p *attendanceBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AttendanceService::FindAll - Begin")

	daoAttendance := p.daoAttendance
	response, err := daoAttendance.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	// Lookup Appuser Info
	p.lookupAppuser(response)

	log.Println("AttendanceService::FindAll - End ")
	return response, nil
}

// ************************
// List - List All records
//
// ************************
func (p *attendanceBaseService) ListNew(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AttendanceService::FindAll - Begin")

	daoAttendance := p.daoAttendance
	response, err := daoAttendance.ListNew(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}
	log.Println("AttendanceService::FindAll - End ")
	return response, nil
}

// *************************
// Get - Get Details
//
// *************************
func (p *attendanceBaseService) Get(appattendance_id string) (utils.Map, error) {
	log.Printf("AttendanceService::FindByCode::  Begin %v", appattendance_id)

	data, err := p.daoAttendance.Get(appattendance_id)
	log.Println("AttendanceService::FindByCode:: End ", err)

	// Lookup Appuser Info
	p.mergeUserInfo(data)
	return data, err
}

// ************************
// Find - Find Service
//
// ************************
func (p *attendanceBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AttendanceService::FindByCode::  Begin ", filter)

	data, err := p.daoAttendance.Find(filter)
	log.Println("AttendanceService::FindByCode:: End ", data, err)
	return data, err
}

// ************************
// Create - Create Service
//
// ************************
// func (p *attendanceBaseService) Create(indata utils.Map) (utils.Map, error) {

// 	log.Println("AttendanceService::Create - Begin")

// 	// Create AttendanceId
// 	attendanceId := utils.GenerateUniqueId("atten")

// 	if utils.IsEmpty(p.staffId) {
// 		err := &utils.AppError{
// 			ErrorCode:   "S30102",
// 			ErrorMsg:    "No StaffId",
// 			ErrorDetail: "No StaffId passed"}
// 		return indata, err
// 	}

// 	indata[hr_common.FLD_ATTENDANCE_ID] = attendanceId
// 	indata[hr_common.FLD_BUSINESS_ID] = p.businessId
// 	indata[hr_common.FLD_STAFF_ID] = p.staffId
// 	indata[hr_common.FLD_DATETIME] = time.Now().UTC() //.Format("2006-01-02 15:04:05")

// 	log.Println("Provided Attendance ID:", attendanceId)

// 	insertResult, err := p.daoAttendance.Create(indata)
// 	log.Println("AttendanceService::Create - End ", insertResult)

// 	return indata, err
// }

// ********************************
// CreateMany - CreateMany Service
//
// ********************************
// func (p *attendanceBaseService) CreateMany(indata utils.Map) (utils.Map, error) {

// 	var err error = nil

// 	log.Println("AttendanceService::CreateMany - Begin")

// 	// Create AttendanceId
// 	attendanceId := utils.GenerateUniqueId("atten")

// 	// Check staffId received in indata
// 	staffId, _ := utils.GetMemberDataStr(indata, hr_common.FLD_STAFF_ID)
// 	if utils.IsEmpty(staffId) {
// 		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "No StaffId", ErrorDetail: "No StaffId passed"}
// 		return indata, err
// 	}
// 	indata[hr_common.FLD_ATTENDANCE_ID] = attendanceId
// 	indata[hr_common.FLD_BUSINESS_ID] = p.businessId

// 	// Convert Date_time string to Date Format
// 	if dataVal, dataOk := indata[hr_common.FLD_DATETIME]; dataOk {
// 		layout := hr_common.DATETIME_PARSE_FORMAT
// 		indata[hr_common.FLD_DATETIME], err = time.Parse(layout, dataVal.(string))
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	insertResult, err := p.daoAttendance.Create(indata)

// 	log.Println("AttendanceService::Create - End ", insertResult)
// 	return indata, err
// }

// *************************
// ClockIn - Clock IN
//
// ************************
func (p *attendanceBaseService) ClockIn(indata utils.Map) (utils.Map, error) {
	var err error = nil

	log.Println("AttendanceService::ClockIn - Begin")

	// Get Timezone Location
	loc, err := p.getTimezoneLocation(indata)
	if err != nil {
		return indata, err
	}

	// Create AttendanceId
	attendanceId := utils.GenerateUniqueId("atten")

	// Add Current DateTime
	// indata[hr_common.FLD_DATETIME] = time.Now().UTC()
	indata[hr_common.FLD_DATETIME] = time.Now().In(loc).Format(time.DateTime)

	// Create ClockIn Data
	var clockIn utils.Map = utils.Map{}

	clockIn[hr_common.FLD_ATTENDANCE_ID] = attendanceId
	clockIn[hr_common.FLD_BUSINESS_ID] = p.businessId
	clockIn[hr_common.FLD_STAFF_ID] = p.staffId

	// Update Clock-In Interface back
	clockIn[hr_common.FLD_CLOCK_IN] = indata

	_, err = p.daoAttendance.Create(clockIn)

	log.Println("AttendanceService::ClockIn - End")
	return clockIn, err
}

// *************************************************
// ClockInMany - Clock In with StaffId and DateTime
//
// ************************************************
func (p *attendanceBaseService) ClockInMany(indata utils.Map) (utils.Map, error) {
	var err error = nil

	log.Println("AttendanceService::ClockInMany - Begin")

	// Create AttendanceId
	attendanceId := utils.GenerateUniqueId("atten")

	// Check staffId received in indata
	staffId, _ := utils.GetMemberDataStr(indata, hr_common.FLD_STAFF_ID)
	_, err = p.daoStaff.Get(staffId)
	if err != nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Invalid StaffId", ErrorDetail: "No such StaffId found"}
		return indata, err
	}

	err = p.validateDateTime(indata)
	if err != nil {
		return nil, err
	}

	// Remove StaffId from indata
	delete(indata, hr_common.FLD_STAFF_ID)

	// Prepare ClockIn Data
	var clockIn utils.Map = utils.Map{}
	clockIn[hr_common.FLD_ATTENDANCE_ID] = attendanceId
	clockIn[hr_common.FLD_BUSINESS_ID] = p.businessId
	clockIn[hr_common.FLD_STAFF_ID] = staffId

	// Update Clock-In Interface back
	clockIn[hr_common.FLD_CLOCK_IN] = indata

	insertResult, err := p.daoAttendance.Create(clockIn)

	log.Println("AttendanceService::ClockInMany - End ", insertResult)
	return clockIn, err

}

// *************************
// ClockOut - Clock Out
//
// ************************
func (p *attendanceBaseService) ClockOut(attendance_id string, indata utils.Map) (utils.Map, error) {
	var err error = nil

	log.Println("AttendanceService::ClockOut - Begin")

	data, err := p.daoAttendance.Get(attendance_id)
	if err != nil {
		return indata, err
	}

	// Get Timezone Location
	loc, err := p.getTimezoneLocation(indata)
	if err != nil {
		return indata, err
	}

	// Update DateTime
	//indata[hr_common.FLD_DATETIME] = time.Now().UTC()
	indata[hr_common.FLD_DATETIME] = time.Now().In(loc).Format(time.DateTime)

	// Update Clock-In Interface back
	data[hr_common.FLD_CLOCK_OUT] = indata

	_, err = p.daoAttendance.Update(attendance_id, data)

	log.Println("AttendanceService::ClockIn - End")
	return data, err
}

// *************************************************
// ClockInMany - Clock Out with AttendanceId and DateTime
//
// ************************************************
func (p *attendanceBaseService) ClockOutMany(indata utils.Map) (utils.Map, error) {
	var err error = nil

	log.Println("AttendanceService::ClockOutMany - Begin")
	// Check staffId received in indata
	attendanceId, _ := utils.GetMemberDataStr(indata, hr_common.FLD_ATTENDANCE_ID)
	data, err := p.daoAttendance.Get(attendanceId)
	if err != nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Invalid AttendanceId", ErrorDetail: "No such AttendanceId found"}
		return nil, err
	}

	err = p.validateDateTime(indata)
	if err != nil {
		return nil, err
	}

	// Remove StaffId from indata
	delete(indata, hr_common.FLD_ATTENDANCE_ID)

	// Update Clock-In Interface back
	data[hr_common.FLD_CLOCK_OUT] = indata

	_, err = p.daoAttendance.Update(attendanceId, data)

	log.Println("AttendanceService::ClockIn - End")
	return data, err

}

// ************************
// Update - Update Service
//
// ************************
func (p *attendanceBaseService) Update(attendance_id string, indata utils.Map) (utils.Map, error) {

	log.Println("AttendanceService::Update - Begin")

	data, err := p.daoAttendance.Get(attendance_id)
	if err != nil {
		return data, err
	}

	// Delete the Key fields
	delete(indata, hr_common.FLD_ATTENDANCE_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)
	delete(indata, hr_common.FLD_STAFF_ID)
	delete(indata, hr_common.FLD_DATETIME)

	clockInData, err := utils.GetMemberData(indata, hr_common.FLD_CLOCK_IN)
	if err == nil {
		err = p.validateDateTime(clockInData.(map[string]interface{}))
		if err != nil {
			log.Println("Failed to Parse clock_in->date_time", err)
			return nil, err
		}
	}

	clockOutData, err := utils.GetMemberData(indata, hr_common.FLD_CLOCK_OUT)
	if err == nil {
		err = p.validateDateTime(clockOutData.(map[string]interface{}))
		if err != nil {
			log.Println("Failed to Parse clock_in->date_time", err)
			return nil, err
		}
	}

	data, err = p.daoAttendance.Update(attendance_id, indata)
	log.Println("AttendanceService::Update - End ")
	return data, err
}

// ************************
// Delete - Delete Service
//
// ************************
func (p *attendanceBaseService) Delete(attendance_id string, delete_permanent bool) error {

	log.Println("AttendanceService::Delete - Begin", attendance_id, delete_permanent)

	daoAttendance := p.daoAttendance
	_, err := daoAttendance.Get(attendance_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoAttendance.Delete(attendance_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoAttendance.Update(attendance_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("AttendanceService::Delete - End")
	return nil
}

// ***********************************************
// DeleteAll - Delete All Attendance for the staff
//
// ***********************************************
func (p *attendanceBaseService) DeleteAll(delete_permanent bool) error {

	log.Println("AttendanceService::DeleteAll - Begin", delete_permanent)

	daoAttendance := p.daoAttendance
	if delete_permanent {
		result, err := daoAttendance.DeleteMany()
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoAttendance.UpdateMany(indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("AttendanceService::DeleteAll - End")
	return nil
}

func (p *attendanceBaseService) errorReturn(err error) (AttendanceService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}

func (p *attendanceBaseService) getTimezoneLocation(indata utils.Map) (*time.Location, error) {
	// Get Timezone Information from
	businessTimezone, err := utils.GetMemberDataStr(indata, business_common.FLD_BUSINESS_TIMEZONE)
	if err != nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "No Timezone", ErrorDetail: "No Timezone Information sent"}
		return nil, err
	}
	// Load Location
	loc, err := time.LoadLocation(businessTimezone)
	if err != nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Invalid Timezone", ErrorDetail: "Timezone Information is invalid"}
		return nil, err
	}

	// Remove Timezone from indata
	delete(indata, business_common.FLD_BUSINESS_TIMEZONE)

	return loc, nil
}

func (p *attendanceBaseService) validateDateTime(indata utils.Map) error {
	var err error = nil

	// Validate given data_time
	dateTime, err := utils.GetMemberDataStr(indata, hr_common.FLD_DATETIME)
	if err == nil {
		_, err = time.Parse(time.DateTime, dateTime)
		if err != nil {
			return err
		}
	}
	return err
}

func (p *attendanceBaseService) lookupAppuser(response utils.Map) {

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

func (p *attendanceBaseService) mergeUserInfo(staffInfo utils.Map) {

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
