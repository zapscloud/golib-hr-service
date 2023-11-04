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

// ClientService - Clients Service structure
type ClientService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(clientId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(clientId string, indata utils.Map) (utils.Map, error)
	Delete(clientId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// ClientBaseService - Clients Service structure
type clientBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoClient           hr_repository.ClientDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               ClientService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewClientService(props utils.Map) (ClientService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("ClientService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := clientBaseService{}
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
	p.daoClient = hr_repository.NewClientDao(p.dbRegion.GetClient(), p.businessID)
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

func (p *clientBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *clientBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("ClientService::FindAll - Begin")

	daoClient := p.daoClient
	response, err := daoClient.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("ClientService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *clientBaseService) Get(clientId string) (utils.Map, error) {
	log.Printf("ClientService::FindByCode::  Begin %v", clientId)

	data, err := p.daoClient.Get(clientId)
	log.Println("ClientService::FindByCode:: End ", err)
	return data, err
}

func (p *clientBaseService) Find(filter string) (utils.Map, error) {
	log.Println("ClientService::FindByCode::  Begin ", filter)

	data, err := p.daoClient.Find(filter)
	log.Println("ClientService::FindByCode:: End ", data, err)
	return data, err
}

func (p *clientBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var clientId string

	dataval, dataok := indata[hr_common.FLD_CLIENT_ID]
	if dataok {
		clientId = strings.ToLower(dataval.(string))
	} else {
		clientId = utils.GenerateUniqueId("clnt")
		log.Println("Unique Client ID", clientId)
	}
	indata[hr_common.FLD_CLIENT_ID] = clientId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Client ID:", clientId)

	_, err := p.daoClient.Get(clientId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Client ID !", ErrorDetail: "Given Client ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoClient.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *clientBaseService) Update(clientId string, indata utils.Map) (utils.Map, error) {

	log.Println("ClientService::Update - Begin")

	data, err := p.daoClient.Get(clientId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_CLIENT_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoClient.Update(clientId, indata)
	log.Println("ClientService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *clientBaseService) Delete(clientId string, delete_permanent bool) error {

	log.Println("ClientService::Delete - Begin", clientId)

	daoClient := p.daoClient
	_, err := daoClient.Get(clientId)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoClient.Delete(clientId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoClient.Update(clientId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("ClientService::Delete - End")
	return nil
}

func (p *clientBaseService) errorReturn(err error) (ClientService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
