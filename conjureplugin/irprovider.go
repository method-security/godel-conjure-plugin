// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

package conjureplugin

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	"github.com/palantir/godel-conjure-plugin/v6/ir-gen-cli-bundler/conjureircli"
	"github.com/palantir/pkg/safehttp"
	"github.com/pkg/errors"
)

type IRProvider interface {
	IRBytes() ([]byte, error)
	// Generated returns true if the IR provided by this provider is generated from YAML, false otherwise.
	GeneratedFromYAML() bool
}

var _ IRProvider = &localYAMLIRProvider{}

type localYAMLIRProvider struct {
	path   string
	params []conjureircli.Param
}

// NewLocalYAMLIRProvider returns an IRProvider that provides IR generated from local YAML. The provided path must be a
// path to a Conjure YAML file or a directory that contains Conjure YAML files.
func NewLocalYAMLIRProvider(path string, params ...conjureircli.Param) IRProvider {
	return &localYAMLIRProvider{
		path:   path,
		params: params,
	}
}

func (p *localYAMLIRProvider) IRBytes() ([]byte, error) {
	return conjureircli.InputPathToIRWithParams(p.path, p.params...)
}

func (p *localYAMLIRProvider) GeneratedFromYAML() bool {
	return true
}

var _ IRProvider = &urlIRProvider{}

type urlIRProvider struct {
	irURL string
}

// NewHTTPIRProvider returns an IRProvider that that provides IR downloaded from the provided URL over HTTP.
func NewHTTPIRProvider(irURL string) IRProvider {
	return &urlIRProvider{
		irURL: irURL,
	}
}

func (p *urlIRProvider) IRBytes() ([]byte, error) {
	resp, cleanup, err := safehttp.Get(http.DefaultClient, p.irURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer cleanup()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("expected response status 200 when fetching IR from remote source %s, but got %d", p.irURL, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func (p *urlIRProvider) GeneratedFromYAML() bool {
	return false
}

var _ IRProvider = &localFileIRProvider{}

type localFileIRProvider struct {
	path string
}

// NewLocalFileIRProvider returns an IRProvider that that provides IR from the local file at the specified path.
func NewLocalFileIRProvider(path string) IRProvider {
	return &localFileIRProvider{
		path: path,
	}
}

func (p *localFileIRProvider) IRBytes() ([]byte, error) {
	return ioutil.ReadFile(p.path)
}

func (p *localFileIRProvider) GeneratedFromYAML() bool {
	return false
}

var _ IRProvider = &codeArtifactIRProvider{}

type codeArtifactIRProvider struct {
	domain       string
	domainOwner  string
	repository   string
	packageGroup string
	packageName  string
	version      string
	region       *string
	profile      *string
}

// NewCodeArtifactIRProvider returns an IRProvider that downloads IR from AWS CodeArtifact.
func NewCodeArtifactIRProvider(domain, domainOwner, repository, packageGroup, packageName, version string, region, profile *string) IRProvider {
	return &codeArtifactIRProvider{
		domain:       domain,
		domainOwner:  domainOwner,
		repository:   repository,
		packageGroup: packageGroup,
		packageName:  packageName,
		version:      version,
		region:       region,
		profile:      profile,
	}
}

func (p *codeArtifactIRProvider) IRBytes() ([]byte, error) {
	ctx := context.Background()

	// Load AWS configuration
	var cfg config.LoadOptionsFunc
	if p.profile != nil {
		cfg = config.WithSharedConfigProfile(*p.profile)
	}
	var regionOpt config.LoadOptionsFunc
	if p.region != nil {
		regionOpt = config.WithRegion(*p.region)
	}

	var opts []func(*config.LoadOptions) error
	if cfg != nil {
		opts = append(opts, cfg)
	}
	if regionOpt != nil {
		opts = append(opts, regionOpt)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS configuration")
	}

	// Create CodeArtifact client
	client := codeartifact.NewFromConfig(awsCfg)

	// Get the package version asset download URL
	getPackageVersionAssetInput := &codeartifact.GetPackageVersionAssetInput{
		Domain:         &p.domain,
		DomainOwner:    &p.domainOwner,
		Repository:     &p.repository,
		Format:         "generic", // CodeArtifact generic format for raw files
		Namespace:      &p.packageGroup,
		Package:        &p.packageName,
		PackageVersion: &p.version,
		Asset:          &p.packageName, // Asset name is typically the same as package name for IR files
	}

	result, err := client.GetPackageVersionAsset(ctx, getPackageVersionAssetInput)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get package version asset from CodeArtifact: domain=%s, repository=%s, package=%s/%s, version=%s",
			p.domain, p.repository, p.packageGroup, p.packageName, p.version)
	}

	// Read the asset content
	assetBytes, err := ioutil.ReadAll(result.Asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read asset content from CodeArtifact response")
	}

	return assetBytes, nil
}

func (p *codeArtifactIRProvider) GeneratedFromYAML() bool {
	return false
}
