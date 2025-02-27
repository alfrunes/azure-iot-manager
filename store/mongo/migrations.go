// Copyright 2021 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package mongo

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
)

const (
	// DbVersion is the current schema version
	DbVersion = "1.0.0"

	// DbName is the database name
	DbName = "azure_iot_manager"
)

// Migrate applies migrations to the database
func Migrate(ctx context.Context,
	db string,
	version string,
	client *mongo.Client,
	automigrate bool) error {
	l := log.FromContext(ctx)

	l.Infof("migrating %s", db)

	ver, err := migrate.NewVersion(version)
	if err != nil {
		return errors.Wrap(err, "failed to parse service version")
	}

	m := migrate.DummyMigrator{
		Client:      client,
		Db:          db,
		Automigrate: automigrate,
	}

	var migrations []migrate.Migration

	err = m.Apply(ctx, *ver, migrations)
	if err != nil {
		return errors.Wrap(err, "failed to apply migrations")
	}

	return nil
}
