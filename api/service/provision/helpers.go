// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provision

import (
	"github.com/globocom/tsuru/api/auth"
	"github.com/globocom/tsuru/api/service"
	"github.com/globocom/tsuru/errors"
	"net/http"
)

func getServiceOrError(name string, u *auth.User) (service.Service, error) {
	s := service.Service{Name: name}
	err := s.Get()
	if err != nil {
		return s, &errors.Http{Code: http.StatusNotFound, Message: "Service not found"}
	}
	if !auth.CheckUserAccess(s.OwnerTeams, u) {
		msg := "This user does not have access to this service"
		return s, &errors.Http{Code: http.StatusForbidden, Message: msg}
	}
	return s, err
}

func servicesAndInstancesByOwner(u *auth.User) []service.ServiceModel {
	services, _ := service.GetServicesByOwnerTeams("owner_teams", u)
	sInstances, _ := service.GetServiceInstancesByServices(services)
	results := make([]service.ServiceModel, len(services))
	for i, s := range services {
		results[i].Service = s.Name
		for _, si := range sInstances {
			if si.ServiceName == s.Name {
				results[i].Instances = append(results[i].Instances, si.Name)
			}
		}
	}
	return results
}
