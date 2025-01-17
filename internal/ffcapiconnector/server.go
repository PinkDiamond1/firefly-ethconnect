// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ffcapiconnector

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Masterminds/semver"
	"github.com/hyperledger/firefly-common/pkg/ffcapi"
	"github.com/hyperledger/firefly-ethconnect/internal/errors"
	"github.com/hyperledger/firefly-ethconnect/internal/eth"
	log "github.com/sirupsen/logrus"
)

var supportedAPIVersions = "1.0.x"

type FFCServer interface {
	ServeFFCAPI(ctx context.Context, payload []byte, w http.ResponseWriter)
}

type FFCServerConf struct {
	eth.EthCommonConf
}

type ffcServer struct {
	rpc          eth.RPCClient
	versionCheck *semver.Constraints
	handlerMap   map[ffcapi.RequestType]ffcHandler

	gasEstimationFactor float64
}

type ffcHandler func(ctx context.Context, payload []byte) (res interface{}, reason ffcapi.ErrorReason, err error)

func NewFFCServer(rpc eth.RPCClient, conf *FFCServerConf) FFCServer {
	s := &ffcServer{
		rpc:                 rpc,
		gasEstimationFactor: conf.GasEstimationFactor,
	}
	s.handlerMap = map[ffcapi.RequestType]ffcHandler{
		ffcapi.RequestTypeCreateBlockListener:  s.createBlockListener,
		ffcapi.RequestTypeExecQuery:            s.execQuery,
		ffcapi.RequestTypeGetBlockInfoByHash:   s.getBlockInfoByHash,
		ffcapi.RequestTypeGetBlockInfoByNumber: s.getBlockInfoByNumber,
		ffcapi.RequestTypeGetGasPrice:          s.getGasPrice,
		ffcapi.RequestTypeGetNewBlockHashes:    s.getNewBlockHashes,
		ffcapi.RequestTypeGetNextNonce:         s.getNextNonce,
		ffcapi.RequestTypeGetReceipt:           s.getReceipt,
		ffcapi.RequestTypePrepareTransaction:   s.prepareTransaction,
		ffcapi.RequestTypeSendTransaction:      s.sendTransaction,
	}
	s.versionCheck, _ = semver.NewConstraint(supportedAPIVersions)
	return s
}

func (s *ffcServer) ServeFFCAPI(ctx context.Context, payload []byte, w http.ResponseWriter) {
	var resBase ffcapi.RequestBase
	_ = json.Unmarshal(payload, &resBase)
	var resBody interface{}
	status := 200
	reason := ffcapi.ErrorReasonInvalidInputs
	handler, err := s.validateHeader(&resBase.FFCAPI)
	if err == nil {
		log.Tracef("--> %s %s", resBase.FFCAPI.RequestType, payload)
		resBody, reason, err = handler(ctx, payload)
		log.Tracef("<-- %s %s %v", resBase.FFCAPI.RequestType, reason, err)
	}
	if err != nil {
		log.Errorf("Request failed: %s", err)
		resBody = &ffcapi.ErrorResponse{Error: err.Error(), Reason: reason}
		status = s.mapReasonStatus(reason)
	}
	w.Header().Set("Content-Type", "application/json")
	resBytes, _ := json.Marshal(resBody)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(resBytes)), 10))
	w.WriteHeader(status)
	_, _ = w.Write(resBytes)
}

func (s *ffcServer) mapReasonStatus(reason ffcapi.ErrorReason) int {
	switch reason {
	case ffcapi.ErrorReasonNotFound:
		return http.StatusNotFound
	case ffcapi.ErrorReasonInvalidInputs:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (s *ffcServer) validateHeader(header *ffcapi.Header) (ffcHandler, error) {
	v, err := semver.NewVersion(string(header.Version))
	if err != nil {
		return nil, errors.Errorf(errors.FFCBadVersion, header.Version, err)
	}
	if !s.versionCheck.Check(v) {
		return nil, errors.Errorf(errors.FFCUnsupportedVersion, header.Version)
	}
	if header.RequestID == nil {
		return nil, errors.Errorf(errors.FFCMissingRequestID)
	}
	handler, ok := s.handlerMap[header.RequestType]
	if !ok {
		return nil, errors.Errorf(errors.FFCUnsupportedRequestType, header.RequestType)
	}
	return handler, nil
}
