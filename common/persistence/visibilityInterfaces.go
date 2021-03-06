// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package persistence

import (
	commonpb "go.temporal.io/temporal-proto/common"
	executionpb "go.temporal.io/temporal-proto/execution"
	"go.temporal.io/temporal-proto/serviceerror"

	"github.com/temporalio/temporal/common/definition"
)

// Interfaces for the Visibility Store.
// This is a secondary store that is eventually consistent with the main
// executions store, and stores workflow execution records for visibility
// purposes.

type (

	// RecordWorkflowExecutionStartedRequest is used to add a record of a newly
	// started execution
	RecordWorkflowExecutionStartedRequest struct {
		NamespaceID        string
		Namespace          string // not persisted, used as config filter key
		Execution          commonpb.WorkflowExecution
		WorkflowTypeName   string
		StartTimestamp     int64
		ExecutionTimestamp int64
		RunTimeout         int64 // not persisted, used for cassandra ttl
		TaskID             int64 // not persisted, used as condition update version for ES
		Memo               *commonpb.Memo
		TaskList           string
		SearchAttributes   map[string]*commonpb.Payload
	}

	// RecordWorkflowExecutionClosedRequest is used to add a record of a newly
	// closed execution
	RecordWorkflowExecutionClosedRequest struct {
		NamespaceID        string
		Namespace          string // not persisted, used as config filter key
		Execution          commonpb.WorkflowExecution
		WorkflowTypeName   string
		StartTimestamp     int64
		ExecutionTimestamp int64
		CloseTimestamp     int64
		Status             executionpb.WorkflowExecutionStatus
		HistoryLength      int64
		RetentionSeconds   int64
		TaskID             int64 // not persisted, used as condition update version for ES
		Memo               *commonpb.Memo
		TaskList           string
		SearchAttributes   map[string]*commonpb.Payload
	}

	// UpsertWorkflowExecutionRequest is used to upsert workflow execution
	UpsertWorkflowExecutionRequest struct {
		NamespaceID        string
		Namespace          string // not persisted, used as config filter key
		Execution          commonpb.WorkflowExecution
		WorkflowTypeName   string
		StartTimestamp     int64
		ExecutionTimestamp int64
		WorkflowTimeout    int64 // not persisted, used for cassandra ttl
		TaskID             int64 // not persisted, used as condition update version for ES
		Memo               *commonpb.Memo
		TaskList           string
		SearchAttributes   map[string]*commonpb.Payload
	}

	// ListWorkflowExecutionsRequest is used to list executions in a namespace
	ListWorkflowExecutionsRequest struct {
		NamespaceID       string
		Namespace         string // namespace name is not persisted, but used as config filter key
		EarliestStartTime int64
		LatestStartTime   int64
		// Maximum number of workflow executions per page
		PageSize int
		// Token to continue reading next page of workflow executions.
		// Pass in empty slice for first page.
		NextPageToken []byte
	}

	// ListWorkflowExecutionsRequestV2 is used to list executions in a namespace
	ListWorkflowExecutionsRequestV2 struct {
		NamespaceID string
		Namespace   string // namespace name is not persisted, but used as config filter key
		PageSize    int    // Maximum number of workflow executions per page
		// Token to continue reading next page of workflow executions.
		// Pass in empty slice for first page.
		NextPageToken []byte
		Query         string
	}

	// ListWorkflowExecutionsResponse is the response to ListWorkflowExecutionsRequest
	ListWorkflowExecutionsResponse struct {
		Executions []*executionpb.WorkflowExecutionInfo
		// Token to read next page if there are more workflow executions beyond page size.
		// Use this to set NextPageToken on ListWorkflowExecutionsRequest to read the next page.
		NextPageToken []byte
	}

	// CountWorkflowExecutionsRequest is request from CountWorkflowExecutions
	CountWorkflowExecutionsRequest struct {
		NamespaceID string
		Namespace   string // namespace name is not persisted, but used as config filter key
		Query       string
	}

	// CountWorkflowExecutionsResponse is response to CountWorkflowExecutions
	CountWorkflowExecutionsResponse struct {
		Count int64
	}

	// ListWorkflowExecutionsByTypeRequest is used to list executions of
	// a specific type in a namespace
	ListWorkflowExecutionsByTypeRequest struct {
		ListWorkflowExecutionsRequest
		WorkflowTypeName string
	}

	// ListWorkflowExecutionsByWorkflowIDRequest is used to list executions that
	// have specific WorkflowID in a namespace
	ListWorkflowExecutionsByWorkflowIDRequest struct {
		ListWorkflowExecutionsRequest
		WorkflowID string
	}

	// ListClosedWorkflowExecutionsByStatusRequest is used to list executions that
	// have specific close status
	ListClosedWorkflowExecutionsByStatusRequest struct {
		ListWorkflowExecutionsRequest
		Status executionpb.WorkflowExecutionStatus
	}

	// GetClosedWorkflowExecutionRequest is used retrieve the record for a specific execution
	GetClosedWorkflowExecutionRequest struct {
		NamespaceID string
		Namespace   string // namespace name is not persisted, but used as config filter key
		Execution   commonpb.WorkflowExecution
	}

	// GetClosedWorkflowExecutionResponse is the response to GetClosedWorkflowExecutionRequest
	GetClosedWorkflowExecutionResponse struct {
		Execution *executionpb.WorkflowExecutionInfo
	}

	// VisibilityDeleteWorkflowExecutionRequest contains the request params for DeleteWorkflowExecution call
	VisibilityDeleteWorkflowExecutionRequest struct {
		NamespaceID string
		RunID       string
		WorkflowID  string
		TaskID      int64
	}

	// VisibilityManager is used to manage the visibility store
	VisibilityManager interface {
		Closeable
		GetName() string
		RecordWorkflowExecutionStarted(request *RecordWorkflowExecutionStartedRequest) error
		RecordWorkflowExecutionClosed(request *RecordWorkflowExecutionClosedRequest) error
		UpsertWorkflowExecution(request *UpsertWorkflowExecutionRequest) error
		ListOpenWorkflowExecutions(request *ListWorkflowExecutionsRequest) (*ListWorkflowExecutionsResponse, error)
		ListClosedWorkflowExecutions(request *ListWorkflowExecutionsRequest) (*ListWorkflowExecutionsResponse, error)
		ListOpenWorkflowExecutionsByType(request *ListWorkflowExecutionsByTypeRequest) (*ListWorkflowExecutionsResponse, error)
		ListClosedWorkflowExecutionsByType(request *ListWorkflowExecutionsByTypeRequest) (*ListWorkflowExecutionsResponse, error)
		ListOpenWorkflowExecutionsByWorkflowID(request *ListWorkflowExecutionsByWorkflowIDRequest) (*ListWorkflowExecutionsResponse, error)
		ListClosedWorkflowExecutionsByWorkflowID(request *ListWorkflowExecutionsByWorkflowIDRequest) (*ListWorkflowExecutionsResponse, error)
		ListClosedWorkflowExecutionsByStatus(request *ListClosedWorkflowExecutionsByStatusRequest) (*ListWorkflowExecutionsResponse, error)
		GetClosedWorkflowExecution(request *GetClosedWorkflowExecutionRequest) (*GetClosedWorkflowExecutionResponse, error)
		DeleteWorkflowExecution(request *VisibilityDeleteWorkflowExecutionRequest) error
		ListWorkflowExecutions(request *ListWorkflowExecutionsRequestV2) (*ListWorkflowExecutionsResponse, error)
		ScanWorkflowExecutions(request *ListWorkflowExecutionsRequestV2) (*ListWorkflowExecutionsResponse, error)
		CountWorkflowExecutions(request *CountWorkflowExecutionsRequest) (*CountWorkflowExecutionsResponse, error)
	}
)

// NewOperationNotSupportErrorForVis create error for operation not support in visibility
func NewOperationNotSupportErrorForVis() error {
	return serviceerror.NewInvalidArgument("Operation not support. Please use on ElasticSearch")
}

// IsNopUpsertWorkflowRequest return whether upsert request should be no-op
func IsNopUpsertWorkflowRequest(request *InternalUpsertWorkflowExecutionRequest) bool {
	_, exist := request.SearchAttributes[definition.TemporalChangeVersion]
	return exist
}
