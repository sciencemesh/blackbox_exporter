// Copyright 2018-2020 CERN
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
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package main

/*
Usage example: $> ./ioplogin -target=reva:19000 -user=yourname -pass=secret -insecure=true
*/

import (
	"flag"
	"fmt"
	"os"

	iop "github.com/cs3org/reva/pkg/sdk"
)

var (
	target, user, pass string
	insecure           bool
)

const (
	checkOK      = 0
	checkWarning = 1
	checkError   = 2
	checkUnknown = 3
)

func init() {
	flag.StringVar(&target, "target", "", "the target iop")
	flag.StringVar(&user, "user", "", "the user name")
	flag.StringVar(&pass, "pass", "", "the user pass")
	flag.BoolVar(&insecure, "insecure", false, "disables grpc transport security")
	flag.Parse()
}

func main() {
	os.Exit(run())
}

func run() int {
	session := iop.MustNewSession() // Panics if this fails (should usually not happen)
	session.Initiate(target, insecure)
	err := session.BasicLogin(user, pass)
	if err != nil {
		// output error to stdout
		fmt.Printf("%v\n", err)
		return checkError
	}
	fmt.Println("IOP login successful")
	return checkOK
}
