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

package payloads

import (
	"bytes"

	commonpb "go.temporal.io/temporal-proto/common"
	"go.temporal.io/temporal/encoded"

	"github.com/temporalio/temporal/common/payload"
)

var (
	dataConverter = encoded.GetDefaultDataConverter()
)

func EncodeString(str string) *commonpb.Payloads {
	// Error can be safely ignored here becase string always can be converted to JSON
	ps, _ := dataConverter.ToData(str)
	return ps
}

func EncodeBytes(bytes []byte) *commonpb.Payloads {
	// Error can be safely ignored here becase []byte always can be raw encoded
	ps, _ := dataConverter.ToData(bytes)
	return ps
}

func Encode(value ...interface{}) (*commonpb.Payloads, error) {
	return dataConverter.ToData(value...)
}

func Decode(ps *commonpb.Payloads, valuePtr ...interface{}) error {
	return dataConverter.FromData(ps, valuePtr...)
}

func ToString(ps *commonpb.Payloads) string {
	var buf bytes.Buffer
	buf.WriteString("{")
	for _, p := range ps.GetPayloads() {
		buf.WriteString(payload.ToString(p))
		buf.WriteString(",")
	}
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1) // cut the last comma
	}
	buf.WriteString("}")
	return buf.String()
}
