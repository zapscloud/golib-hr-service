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

// PositionService - Accounts Service structure
type PositionService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(position_id string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(position_id string, indata utils.Map) (utils.Map, error)
	Delete(position_id string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// positionBaseService - Accounts Service structure
type positionBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoPosition         hr_repository.PositionDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               PositionService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewPositionService(props utils.Map) (PositionService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("PositionService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := positionBaseService{}

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
	p.daoPosition = hr_repository.NewPositionDao(p.dbRegion.GetClient(), p.businessID)
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

func (p *positionBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *positionBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoPosition := p.daoPosition
	response, err := daoPosition.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *positionBaseService) Get(position_id string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", position_id)

	data, err := p.daoPosition.Get(position_id)
	log.Println("AccountService::FindByCode:: End ", err)
	return data, err
}

func (p *positionBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoPosition.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *positionBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")
	var posId string

	dataval, dataok := indata[hr_common.FLD_POSITION_ID]
	if dataok {
		posId = strings.ToLower(dataval.(string))
	} else {
		posId = utils.GenerateUniqueId("posi")
		log.Println("Unique Account ID", posId)
	}
	indata[hr_common.FLD_POSITION_ID] = posId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Account ID:", posId)

	_, err := p.daoPosition.Get(posId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Account ID !", ErrorDetail: "Given Account ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoPosition.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *positionBaseService) Update(position_id string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoPosition.Get(position_id)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_POSITION_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoPosition.Update(position_id, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *positionBaseService) Delete(position_id string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", position_id)

	daoPosition := p.daoPosition
	_, err := daoPosition.Get(position_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoPosition.Delete(position_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoPosition.Update(position_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("PositionService::Delete - End")
	return nil
}

func (p *positionBaseService) errorReturn(err error) (PositionService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
