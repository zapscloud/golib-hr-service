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

// ShiftProfileService - Accounts Service structure
type ShiftProfileService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(shiftProfileId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(shiftProfileId string, indata utils.Map) (utils.Map, error)
	Delete(shiftProfileId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// shiftProfileBaseService - Accounts Service structure
type shiftProfileBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoShift            hr_repository.ShiftProfileDao
	daoPlatformBusiness platform_repository.BusinessDao

	child      ShiftProfileService
	businessId string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewShiftProfileService(props utils.Map) (ShiftProfileService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("ShiftProfileService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := shiftProfileBaseService{}

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

	// Assign the BusinessId & StaffId
	p.businessId = businessId

	// Instantiate other services
	p.daoShift = hr_repository.NewShiftProfileDao(p.dbRegion.GetClient(), p.businessId)
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())

	_, err = p.daoPlatformBusiness.Get(p.businessId)
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

func (p *shiftProfileBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *shiftProfileBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("ShiftProfileService::FindAll - Begin")

	daoShift := p.daoShift
	response, err := daoShift.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("ShiftProfileService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *shiftProfileBaseService) Get(shiftProfileId string) (utils.Map, error) {
	log.Printf("ShiftProfileService::FindByCode::  Begin %v", shiftProfileId)

	data, err := p.daoShift.Get(shiftProfileId)
	log.Println("ShiftProfileService::FindByCode:: End ", err)
	return data, err
}

func (p *shiftProfileBaseService) Find(filter string) (utils.Map, error) {
	log.Println("ShiftProfileService::FindByCode::  Begin ", filter)

	data, err := p.daoShift.Find(filter)
	log.Println("ShiftProfileService::FindByCode:: End ", data, err)
	return data, err
}

func (p *shiftProfileBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var shiftProfileId string

	dataval, dataok := indata[hr_common.FLD_SHIFT_PROFILE_ID]
	if dataok {
		shiftProfileId = strings.ToLower(dataval.(string))
	} else {
		shiftProfileId = utils.GenerateUniqueId("sftprof")
		log.Println("Unique Account ID", shiftProfileId)
	}
	indata[hr_common.FLD_SHIFT_PROFILE_ID] = shiftProfileId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessId
	log.Println("Provided Account ID:", shiftProfileId)

	_, err := p.daoShift.Get(shiftProfileId)
	if err == nil {
		err := &utils.AppError{
			ErrorCode:   "S30102",
			ErrorMsg:    "Existing Shift ID !",
			ErrorDetail: "Given Shift ID already exist"}
		return indata, err
	}
	// // Validate TimeFormat
	// if p.validateTimeFormat(indata) != nil {
	// 	return indata, err
	// }

	insertResult, err := p.daoShift.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *shiftProfileBaseService) Update(shiftProfileId string, indata utils.Map) (utils.Map, error) {

	log.Println("ShiftProfileService::Update - Begin")

	data, err := p.daoShift.Get(shiftProfileId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_SHIFT_PROFILE_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	// // Validate the TimeFormat
	// if p.validateTimeFormat(indata) != nil {
	// 	log.Println("ShiftProfileService::convertStrToTimeFormat - Error ", err)
	// 	return indata, err
	// }

	data, err = p.daoShift.Update(shiftProfileId, indata)
	log.Println("ShiftProfileService::Update - End ", err)
	return data, err
}

// Delete - Delete Service
func (p *shiftProfileBaseService) Delete(shiftProfileId string, delete_permanent bool) error {

	log.Println("ShiftProfileService::Delete - Begin", shiftProfileId)

	daoShift := p.daoShift
	if delete_permanent {
		result, err := daoShift.Delete(shiftProfileId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := p.Update(shiftProfileId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("ShiftProfileService::Delete - End")
	return nil
}

func (p *shiftProfileBaseService) errorReturn(err error) (ShiftProfileService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}

// func (p *shiftProfileBaseService) validateTimeFormat(indata utils.Map) error {
// 	// Convert Time string to Date Format
// 	shiftFromTime, err := utils.GetMemberDataStr(indata, hr_common.FLD_SHIFT_FROM)
// 	if err == nil {
// 		_, err = time.Parse(time.TimeOnly, shiftFromTime)
// 		if err != nil {
// 			err = &utils.AppError{
// 				ErrorCode:   "S30102",
// 				ErrorMsg:    "Failed to Parse Time Value",
// 				ErrorDetail: "Invalid From-Shift-Time value"}
// 			return err
// 		}
// 	}

// 	// Convert Time string to Date Format
// 	shiftToTime, err := utils.GetMemberDataStr(indata, hr_common.FLD_SHIFT_TO)
// 	if err == nil {
// 		_, err = time.Parse(time.TimeOnly, shiftToTime)
// 		if err != nil {
// 			err = &utils.AppError{
// 				ErrorCode:   "S30102",
// 				ErrorMsg:    "Failed to Parse Time Value",
// 				ErrorDetail: "Invalid To-Shift-Time value"}
// 			return err
// 		}
// 	}

// 	return nil
// }
