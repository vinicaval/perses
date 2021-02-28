// Copyright 2021 Amadeus s.a.s
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package project

import (
	"fmt"
	"time"

	"github.com/perses/common/etcd"
	"github.com/perses/perses/internal/api/interface/v1/project"
	"github.com/perses/perses/internal/api/shared"
	v1 "github.com/perses/perses/pkg/model/api/v1"
	"github.com/sirupsen/logrus"
)

type service struct {
	project.Service
	dao project.DAO
}

func NewService(dao project.DAO) project.Service {
	return &service{
		dao: dao,
	}
}

func (s *service) Create(entity interface{}) (interface{}, error) {
	if projectObject, ok := entity.(*v1.Project); ok {
		return s.create(projectObject)
	}
	return nil, fmt.Errorf("wrong entity format, attempting project format, received '%T'", entity)
}

func (s *service) create(entity *v1.Project) (*v1.Project, error) {
	// Update the time contains in the entity
	entity.Metadata.CreateNow()
	if err := s.dao.Create(entity); err != nil {
		if etcd.IsKeyConflict(err) {
			logrus.Debugf("unable to create the project '%s'. It already exits", entity.Metadata.Name)
			return nil, shared.ConflictError
		}
		logrus.WithError(err).Errorf("unable to perform the creation of the project '%s', something wrong with etcd", entity.Metadata.Name)
		return nil, shared.InternalError
	}
	return entity, nil
}

func (s *service) Update(entity interface{}, parameters shared.Parameters) (interface{}, error) {
	if projectObject, ok := entity.(*v1.Project); ok {
		return s.update(projectObject, parameters)
	}
	return nil, fmt.Errorf("wrong entity format, attempting project format, received '%T'", entity)
}

func (s *service) update(entity *v1.Project, parameters shared.Parameters) (*v1.Project, error) {
	if entity.Metadata.Name != parameters.Name {
		logrus.Debugf("name in project '%s' and coming from the http request: '%s' doesn't match", entity.Metadata.Name, parameters.Name)
		return nil, fmt.Errorf("%w: metadata.name and the name in the http path request doesn't match", shared.BadRequestError)
	}
	// find the previous version of the project
	oldEntity, err := s.dao.Get(entity.Metadata.Name)
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			logrus.Debugf("unable to find the project '%s'", entity.Metadata.Name)
			return nil, shared.NotFoundError
		}
		logrus.WithError(err).Errorf("unable to find the previous version of the project '%s', something wrong with etcd", entity.Metadata.Name)
		return nil, shared.InternalError
	}
	// update the immutable field of the newEntity with the old one
	entity.Metadata.CreatedAt = oldEntity.Metadata.CreatedAt
	// update the field UpdatedAt with the new time
	entity.Metadata.UpdatedAt = time.Now().UTC()
	if err := s.dao.Update(entity); err != nil {
		logrus.WithError(err).Errorf("unable to perform the update of the project '%s', something wrong with etcd", entity.Metadata.Name)
		return nil, shared.InternalError
	}
	return entity, nil
}
