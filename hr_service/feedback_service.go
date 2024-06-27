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

// FeedbackService - Feedbacks Service structure
type FeedbackService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(feedbackid string) (utils.Map, error)

	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(feedbackid string, indata utils.Map) (utils.Map, error)
	Delete(feedbackid string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

type feedbackBaseService struct {
	db_utils.DatabaseService
	dbRegion    db_utils.DatabaseService
	daoFeedback hr_repository.FeedbackDao
	daoBusiness platform_repository.BusinessDao
	child       FeedbackService
	businessID  string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewFeedbackService(props utils.Map) (FeedbackService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("FeedbackService::Start ")
	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := feedbackBaseService{}
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
	p.initializeService()

	_, err = p.daoBusiness.Get(businessId)
	if err != nil {
		err := &utils.AppError{
			ErrorCode:   funcode + "01",
			ErrorMsg:    "Invalid business_id",
			ErrorDetail: "Given app_business_id is not exist"}
		return p.errorReturn(err)
	}

	p.child = &p

	return &p, err
}

func (p *feedbackBaseService) EndService() {
	log.Printf("EndFeedbackMongoService ")
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

func (p *feedbackBaseService) initializeService() {
	log.Printf("FeedbackMongoService:: GetBusinessDao ")
	p.daoFeedback = hr_repository.NewFeedbackDao(p.dbRegion.GetClient(), p.businessID)
	p.daoBusiness = platform_repository.NewBusinessDao(p.GetClient())
}

// List - List All records
func (p *feedbackBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("FeedbackService::FindAll - Begin")

	daoFeedback := p.daoFeedback
	response, err := daoFeedback.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("FeedbackService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *feedbackBaseService) Get(feedback_id string) (utils.Map, error) {
	log.Printf("FeedbackService::FindByCode::  Begin %v", feedback_id)

	data, err := p.daoFeedback.Get(feedback_id)
	log.Println("FeedbackService::FindByCode:: End ", err)
	return data, err
}

func (p *feedbackBaseService) Find(filter string) (utils.Map, error) {
	log.Println("FeedbackService::FindByCode::  Begin ", filter)

	data, err := p.daoFeedback.Find(filter)
	log.Println("FeedbackService::FindByCode:: End ", data, err)
	return data, err
}

func (p *feedbackBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")
	var deptId string

	dataval, dataok := indata[hr_common.FLD_FEEDBACK_ID]
	if dataok {
		deptId = strings.ToLower(dataval.(string))
	} else {
		deptId = utils.GenerateUniqueId("fedback")
		log.Println("Unique Feedback ID", deptId)
	}

	indata[hr_common.FLD_FEEDBACK_ID] = deptId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Feedback ID:", dataval)

	_, err := p.daoFeedback.Get(deptId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Feedback ID !", ErrorDetail: "Given Feedback ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoFeedback.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *feedbackBaseService) Update(feedback_id string, indata utils.Map) (utils.Map, error) {

	log.Println("FeedbackService::Update - Begin")

	data, err := p.daoFeedback.Get(feedback_id)
	if err != nil {
		return data, err
	}
	// Delete unique fields
	delete(indata, hr_common.FLD_BUSINESS_ID)
	delete(indata, hr_common.FLD_FEEDBACK_ID)

	data, err = p.daoFeedback.Update(feedback_id, indata)
	log.Println("FeedbackService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *feedbackBaseService) Delete(feedback_id string, delete_permanent bool) error {

	log.Println("FeedbackService::Delete - Begin", feedback_id, delete_permanent)

	daoFeedback := p.daoFeedback
	_, err := daoFeedback.Get(feedback_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoFeedback.Delete(feedback_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoFeedback.Update(feedback_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("FeedbackService::Delete - End")
	return nil
}

func (p *feedbackBaseService) errorReturn(err error) (FeedbackService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
