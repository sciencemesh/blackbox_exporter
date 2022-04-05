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

package siteacc

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sciencemesh/blackbox_exporter/siteacc/config"
	"github.com/sciencemesh/blackbox_exporter/siteacc/temp"
)

type queryResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Site *temp.Site `json:"site"`
	} `json:"data"`
}

var (
	siteAccConfig config.SiteAccountsService
)

func SetSiteAccountsServiceConfig(siteacc *config.SiteAccountsService) {
	siteAccConfig = *siteacc
}

func QuerySiteTestUserCredentials(site string) (*temp.Site, error) {
	if siteAccConfig.URL == "" {
		return nil, errors.Errorf("no site accounts service URL configured")
	}
	if siteAccConfig.Security.CredentialsPassphrase == "" {
		return nil, errors.Errorf("no site accounts credentials passphrase configured")
	}

	url, err := GenerateURL(siteAccConfig.URL, "site-get", URLParams{"site": site})
	if err != nil {
		return nil, errors.Wrap(err, "error while generating endpoint URL")
	}

	resp, err := ReadEndpoint(url, &BasicAuth{siteAccConfig.Authentication.Username, siteAccConfig.Authentication.Password}, true)
	if err != nil {
		return nil, errors.Wrap(err, "error while reading site accounts endpoint")
	}

	var data queryResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal response")
	}
	if !data.Success {
		return nil, errors.Errorf("invalid response received")
	}

	id, secret, err := data.Data.Site.Config.TestClientCredentials.Get(siteAccConfig.Security.CredentialsPassphrase)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decrypt test client credentials")
	}
	data.Data.Site.Config.TestClientCredentials.ID = id
	data.Data.Site.Config.TestClientCredentials.Secret = secret

	return data.Data.Site, nil
}
