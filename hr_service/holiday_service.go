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

// HolidayService - Accounts Service structure
type HolidayService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(holiday_id string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(holiday_id string, indata utils.Map) (utils.Map, error)
	Delete(holiday_id string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// holidayBaseService - Accounts Service structure
type holidayBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoHoliday          hr_repository.HolidayDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               HolidayService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewHolidayService(props utils.Map) (HolidayService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("HolidayService::Start")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := holidayBaseService{}

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
	p.daoHoliday = hr_repository.NewHolidayDao(p.dbRegion.GetClient(), p.businessID)
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

func (p *holidayBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *holidayBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoHoliday := p.daoHoliday
	response, err := daoHoliday.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *holidayBaseService) Get(holiday_id string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", holiday_id)

	data, err := p.daoHoliday.Get(holiday_id)
	log.Println("AccountService::FindByCode:: End ", err)
	return data, err
}

func (p *holidayBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoHoliday.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *holidayBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var holidayId string

	dataval, dataok := indata[hr_common.FLD_HOLIDAY_ID]
	if dataok {
		holidayId = strings.ToLower(dataval.(string))
	} else {
		holidayId = utils.GenerateUniqueId("holi")
		log.Println("Unique Account ID", holidayId)
	}
	indata[hr_common.FLD_HOLIDAY_ID] = holidayId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Account ID:", holidayId)

	_, err := p.daoHoliday.Get(holidayId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Account ID !", ErrorDetail: "Given Account ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoHoliday.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *holidayBaseService) Update(holiday_id string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoHoliday.Get(holiday_id)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_HOLIDAY_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoHoliday.Update(holiday_id, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *holidayBaseService) Delete(holiday_id string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", holiday_id)

	daoHoliday := p.daoHoliday
	_, err := daoHoliday.Get(holiday_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoHoliday.Delete(holiday_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoHoliday.Update(holiday_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("HolidayService::Delete - End")
	return nil
}

func (p *holidayBaseService) errorReturn(err error) (HolidayService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
