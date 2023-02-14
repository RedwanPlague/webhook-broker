package controllers

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/newscred/webhook-broker/storage"
	"github.com/newscred/webhook-broker/storage/data"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

var (
	deliveryJobRepo      storage.DeliveryJobRepository
	jobTestChannel       *data.Channel
	jobTestProducer      *data.Producer
	jobTestConsumer      *data.Consumer
	jobTestOtherConsumer *data.Consumer
)

const (
	jobTestChannelID       = "job-test-channel-id"
	jobTestProducerID      = "job-test-producer-id"
	jobTestConsumerID      = "job-test-consumer-id"
	jobPostTestContentType = "application/json"
)

// JobTestSetup is called from TestMain for the package
func JobTestSetup() {
	setupTestChannel()
	setupTestProducer()
	jobTestConsumer = setupTestConsumer(jobTestConsumerID)
	jobTestOtherConsumer = setupTestConsumer("other-" + jobTestConsumerID)

	deliveryJobRepo = storage.NewDeliveryJobRepository(db, messageRepo, consumerRepo)

	for index := 0; index < 50; index++ {
		indexString := strconv.Itoa(index)
		message, err := data.NewMessage(jobTestChannel, jobTestProducer, "payload "+indexString, "type")
		if err != nil {
			log.Fatal()
		}
		message.Priority = randomPriority()
		messageRepo.Create(message)
		job, err := data.NewDeliveryJob(message, jobTestConsumer)
		if err != nil {
			log.Fatal()
		}
		deliveryJobRepo.DispatchMessage(message, job)
	}
}

func setupTestChannel() {
	channel, err := data.NewChannel(jobTestChannelID, successfulGetTestToken)
	if err != nil {
		log.Fatal()
	}
	jobTestChannel, err = channelRepo.Store(channel)
	if err != nil {
		log.Fatal().Err(err)
	}
}

func setupTestProducer() {
	producer, err := data.NewProducer(jobTestProducerID, successfulGetTestToken)
	if err != nil {
		log.Fatal()
	}
	jobTestProducer, err = producerRepo.Store(producer)
	if err != nil {
		log.Fatal().Err(err)
	}
}

func setupTestConsumer(consumerID string) *data.Consumer {
	callbackURL, err := url.Parse("https://imytech.net/")
	if err != nil {
		log.Fatal().Err(err)
	}
	consumer, err := data.NewConsumer(jobTestChannel, consumerID, successfulGetTestToken, callbackURL, data.PullConsumerStr)
	if err != nil {
		log.Fatal()
	}
	consumer, err = consumerRepo.Store(consumer)
	if err != nil {
		log.Fatal().Err(err)
	}
	return consumer
}

func setupTestJob() (*data.DeliveryJob, error) {
	message, err := data.NewMessage(jobTestChannel, jobTestProducer, "payload", "type")
	if err != nil {
		log.Fatal()
		return nil, err
	}
	messageRepo.Create(message)
	job, err := data.NewDeliveryJob(message, jobTestConsumer)
	if err != nil {
		log.Fatal()
		return nil, err
	}
	deliveryJobRepo.DispatchMessage(message, job)
	return job, nil
}

func randomPriority() uint {
	return uint(1 + rand.Intn(10))
}

func TestJobFormatAsRelativeLink(t *testing.T) {
	t.Parallel()
	controller := NewJobController(channelRepo, consumerRepo, deliveryJobRepo)
	assert.Equal(t, jobPath, controller.GetPath())
	formattedPath := controller.FormatAsRelativeLink(
		httprouter.Param{Key: channelIDPathParamKey, Value: "someChannelId"},
		httprouter.Param{Key: consumerIDPathParamKey, Value: "someConsumerId"},
		httprouter.Param{Key: jobIDPathParamKey, Value: "someJobId"},
	)
	assert.Equal(t, "/channel/someChannelId/consumer/someConsumerId/job/someJobId", formattedPath)
}

func TestJobControllerPost_Success(t *testing.T) {
	t.Parallel()
	job, err := setupTestJob()
	assert.NoError(t, err)

	jobController := NewJobController(channelRepo, consumerRepo, deliveryJobRepo)
	testRouter := createTestRouter(jobController)
	testURI := jobController.FormatAsRelativeLink(
		httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
		httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestConsumer.ConsumerID},
		httprouter.Param{Key: jobIDPathParamKey, Value: job.ID.String()},
	)
	t.Log(testURI)

	validTransitions := []struct {
		current data.JobStatus
		next    data.JobStatus
	}{
		{current: data.JobQueued, next: data.JobQueued},
		{current: data.JobQueued, next: data.JobInflight},
		{current: data.JobInflight, next: data.JobInflight},
		{current: data.JobInflight, next: data.JobDead},
		{current: data.JobDead, next: data.JobDead},
		{current: data.JobDead, next: data.JobInflight},
		{current: data.JobInflight, next: data.JobDelivered},
		{current: data.JobDelivered, next: data.JobDelivered},
	}
	for _, transition := range validTransitions {
		testName := "Success:202-Accepted " + transition.current.String() + " to " + transition.next.String()
		t.Run(testName, func(t *testing.T) {
			bodyString := "{\"NextState\": \"" + transition.next.String() + "\"}"
			requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
			req, err := http.NewRequest("POST", testURI, requestBody)
			assert.NoError(t, err)

			req.Header.Add(headerContentType, jobPostTestContentType)
			req.Header.Add(headerChannelToken, jobTestChannel.Token)
			req.Header.Add(headerConsumerToken, jobTestConsumer.Token)

			rr := httptest.NewRecorder()
			testRouter.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusAccepted, rr.Code)

			updatedJob, err := deliveryJobRepo.GetByID(job.ID.String())
			assert.NoError(t, err)
			assert.Equal(t, transition.next, updatedJob.Status)
		})

	}
}

func TestJobsFormatAsRelativeLink(t *testing.T) {
	t.Parallel()
	controller := NewJobsController(consumerRepo, deliveryJobRepo)
	assert.Equal(t, jobsPath, controller.GetPath())
	formattedPath := controller.FormatAsRelativeLink(
		httprouter.Param{Key: channelIDPathParamKey, Value: "someChannelId"},
		httprouter.Param{Key: consumerIDPathParamKey, Value: "someConsumerId"},
	)
	assert.Equal(t, "/channel/someChannelId/consumer/someConsumerId/queued-jobs", formattedPath)
}

func TestJobsControllerGet(t *testing.T) {
	t.Parallel()
	jobsController := NewJobsController(consumerRepo, deliveryJobRepo)
	testRouter := createTestRouter(jobsController)
	testURI := jobsController.FormatAsRelativeLink(
		httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
		httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestConsumer.ConsumerID},
	)
	t.Log(testURI)
	req, _ := http.NewRequest("GET", testURI, nil)
	rr := httptest.NewRecorder()
	testRouter.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	t.Log(body)
	jobListResult := &JobListResult{}
	json.NewDecoder(strings.NewReader(body)).Decode(jobListResult)

	jobs := jobListResult.Result
	assert.Equal(t, 25, len(jobs))

	assert.True(t, sort.SliceIsSorted(jobs, func(i, j int) bool {
		return jobs[i].Message.Priority > jobs[j].Message.Priority
	}))
}

func TestJobControllerPost_TransitionFailure(t *testing.T) {
	t.Parallel()
	job, err := setupTestJob()
	assert.NoError(t, err)

	jobController := NewJobController(channelRepo, consumerRepo, deliveryJobRepo)
	testRouter := createTestRouter(jobController)
	testURI := jobController.FormatAsRelativeLink(
		httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
		httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestConsumer.ConsumerID},
		httprouter.Param{Key: jobIDPathParamKey, Value: job.ID.String()},
	)
	t.Log(testURI)

	runInvalidTransitionTest := func(invalidNextState data.JobStatus) {
		testName := "400 Bad Request " + job.Status.String() + " to " + invalidNextState.String()
		t.Run(testName, func(t *testing.T) {
			bodyString := "{\"NextState\": \"" + invalidNextState.String() + "\"}"
			requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
			req, err := http.NewRequest("POST", testURI, requestBody)
			assert.NoError(t, err)

			req.Header.Add(headerContentType, jobPostTestContentType)
			req.Header.Add(headerChannelToken, jobTestChannel.Token)
			req.Header.Add(headerConsumerToken, jobTestConsumer.Token)

			rr := httptest.NewRecorder()
			testRouter.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)

			updatedJob, err := deliveryJobRepo.GetByID(job.ID.String())
			assert.NoError(t, err)
			assert.Equal(t, job.Status, updatedJob.Status)
		})
	}

	invalidNextStates := []data.JobStatus{data.JobDelivered, data.JobDead}
	for _, invalidNextState := range invalidNextStates {
		runInvalidTransitionTest(invalidNextState)
	}

	jobController.DeliveryJobRepo.MarkJobInflight(job)
	invalidNextStates = []data.JobStatus{data.JobQueued}
	for _, invalidNextState := range invalidNextStates {
		runInvalidTransitionTest(invalidNextState)
	}

	jobController.DeliveryJobRepo.MarkJobDead(job)
	invalidNextStates = []data.JobStatus{data.JobQueued, data.JobDelivered}
	for _, invalidNextState := range invalidNextStates {
		runInvalidTransitionTest(invalidNextState)
	}

	jobController.DeliveryJobRepo.MarkDeadJobAsInflight(job)
	jobController.DeliveryJobRepo.MarkJobDelivered(job)
	invalidNextStates = []data.JobStatus{data.JobQueued, data.JobInflight, data.JobDead}
	for _, invalidNextState := range invalidNextStates {
		runInvalidTransitionTest(invalidNextState)
	}
}

func TestJobControllerPost_Error(t *testing.T) {
	jobController := NewJobController(channelRepo, consumerRepo, deliveryJobRepo)
	testRouter := createTestRouter(jobController)
	t.Run("400 Channel Not Found", func(t *testing.T) {
		t.Parallel()
		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: "invalid-channel-id"},
			httprouter.Param{Key: consumerIDPathParamKey, Value: "invalid-consumer-id"},
			httprouter.Param{Key: jobIDPathParamKey, Value: "invalid-job-id"},
		)
		t.Log(testURI)

		bodyString := "{\"NextState\": \"INFLIGHT\"}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("403 Channel Token Mismatch", func(t *testing.T) {
		t.Parallel()
		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
			httprouter.Param{Key: consumerIDPathParamKey, Value: "invalid-consumer-id"},
			httprouter.Param{Key: jobIDPathParamKey, Value: "invalid-job-id"},
		)
		t.Log(testURI)

		bodyString := "{\"NextState\": \"INFLIGHT\"}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		req.Header.Add(headerContentType, jobPostTestContentType)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("401 Consumer Not Found", func(t *testing.T) {
		t.Parallel()
		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
			httprouter.Param{Key: consumerIDPathParamKey, Value: "invalid-consumer-id"},
			httprouter.Param{Key: jobIDPathParamKey, Value: "invalid-job-id"},
		)
		t.Log(testURI)

		bodyString := "{\"NextState\": \"INFLIGHT\"}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		req.Header.Add(headerContentType, jobPostTestContentType)
		req.Header.Add(headerChannelToken, jobTestChannel.Token)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("403 Consumer Token Mismatch", func(t *testing.T) {
		t.Parallel()
		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
			httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestConsumer.ConsumerID},
			httprouter.Param{Key: jobIDPathParamKey, Value: "invalid-job-id"},
		)
		t.Log(testURI)

		bodyString := "{\"NextState\": \"INFLIGHT\"}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		req.Header.Add(headerContentType, jobPostTestContentType)
		req.Header.Add(headerChannelToken, jobTestChannel.Token)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("404 Job NotFound", func(t *testing.T) {
		t.Parallel()
		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
			httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestConsumer.ConsumerID},
			httprouter.Param{Key: jobIDPathParamKey, Value: "invalid-job-id"},
		)
		t.Log(testURI)

		bodyString := "{\"NextState\": \"INFLIGHT\"}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		req.Header.Add(headerContentType, jobPostTestContentType)
		req.Header.Add(headerChannelToken, jobTestChannel.Token)
		req.Header.Add(headerConsumerToken, jobTestConsumer.Token)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("401 Job Not for Consumer", func(t *testing.T) {
		t.Parallel()
		job, err := setupTestJob()
		assert.NoError(t, err)

		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
			httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestOtherConsumer.ConsumerID},
			httprouter.Param{Key: jobIDPathParamKey, Value: job.ID.String()},
		)
		t.Log(testURI)

		bodyString := "{\"NextState\": \"INFLIGHT\"}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		req.Header.Add(headerContentType, jobPostTestContentType)
		req.Header.Add(headerChannelToken, jobTestChannel.Token)
		req.Header.Add(headerConsumerToken, jobTestConsumer.Token)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("400 Invalid Request Body", func(t *testing.T) {
		t.Parallel()
		job, err := setupTestJob()
		assert.NoError(t, err)

		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
			httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestConsumer.ConsumerID},
			httprouter.Param{Key: jobIDPathParamKey, Value: job.ID.String()},
		)
		t.Log(testURI)

		bodyString := "{}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		req.Header.Add(headerContentType, jobPostTestContentType)
		req.Header.Add(headerChannelToken, jobTestChannel.Token)
		req.Header.Add(headerConsumerToken, jobTestConsumer.Token)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 Invalid NextState", func(t *testing.T) {
		t.Parallel()
		job, err := setupTestJob()
		assert.NoError(t, err)

		testURI := jobController.FormatAsRelativeLink(
			httprouter.Param{Key: channelIDPathParamKey, Value: jobTestChannel.ChannelID},
			httprouter.Param{Key: consumerIDPathParamKey, Value: jobTestConsumer.ConsumerID},
			httprouter.Param{Key: jobIDPathParamKey, Value: job.ID.String()},
		)
		t.Log(testURI)

		bodyString := "{\"NextState\": \"invalid-state\"}"
		requestBody := ioutil.NopCloser(strings.NewReader(bodyString))
		req, err := http.NewRequest("POST", testURI, requestBody)
		assert.NoError(t, err)

		req.Header.Add(headerContentType, jobPostTestContentType)
		req.Header.Add(headerChannelToken, jobTestChannel.Token)
		req.Header.Add(headerConsumerToken, jobTestConsumer.Token)

		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
