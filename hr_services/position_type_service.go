package hr_services

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

// PositionTypeService - Accounts Service structure
type PositionTypeService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(positionTypeId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(positionTypeId string, indata utils.Map) (utils.Map, error)
	Delete(positionTypeId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// positionTypeBaseService - Accounts Service structure
type positionTypeBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoPositionType     hr_repository.PositionTypeDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               PositionTypeService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewPositionTypeService(props utils.Map) (PositionTypeService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("PositionTypeService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := positionTypeBaseService{}

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
	p.daoPositionType = hr_repository.NewPositionTypeDao(p.dbRegion.GetClient(), p.businessID)
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

func (p *positionTypeBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *positionTypeBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoPositionType := p.daoPositionType
	response, err := daoPositionType.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *positionTypeBaseService) Get(positionTypeId string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", positionTypeId)

	data, err := p.daoPositionType.Get(positionTypeId)
	log.Println("AccountService::FindByCode:: End ", err)
	return data, err
}

func (p *positionTypeBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoPositionType.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *positionTypeBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")
	var posTypeId string

	dataval, dataok := indata[hr_common.FLD_POSITION_TYPE_ID]
	if dataok {
		posTypeId = strings.ToLower(dataval.(string))
	} else {
		posTypeId = utils.GenerateUniqueId("posityp")
		log.Println("Unique Account ID", posTypeId)
	}
	indata[hr_common.FLD_POSITION_TYPE_ID] = posTypeId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Account ID:", posTypeId)

	_, err := p.daoPositionType.Get(posTypeId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Account ID !", ErrorDetail: "Given Account ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoPositionType.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *positionTypeBaseService) Update(positionTypeId string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoPositionType.Get(positionTypeId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_POSITION_TYPE_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoPositionType.Update(positionTypeId, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *positionTypeBaseService) Delete(positionTypeId string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", positionTypeId)

	daoPositionType := p.daoPositionType
	_, err := daoPositionType.Get(positionTypeId)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoPositionType.Delete(positionTypeId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoPositionType.Update(positionTypeId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("PositionTypeService::Delete - End")
	return nil
}

func (p *positionTypeBaseService) errorReturn(err error) (PositionTypeService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
