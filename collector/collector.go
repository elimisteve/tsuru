// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/globocom/tsuru/app"
	"github.com/globocom/tsuru/db"
	"github.com/globocom/tsuru/log"
	"github.com/globocom/tsuru/provision"
	"labix.org/v2/mgo/bson"
	"sort"
)

// AppList is a list of apps. It's not thread safe.
type AppList []*app.App

func (l AppList) Search(name string) (*app.App, int) {
	index := sort.Search(len(l), func(i int) bool {
		return l[i].Name >= name
	})
	if index < len(l) && l[index].Name == name {
		return l[index], -1
	} else if index < len(l) {
		return &app.App{Name: name}, index
	}
	return &app.App{Name: name}, len(l)
}

func (l *AppList) Add(a *app.App, index int) {
	length := len(*l)
	*l = append(*l, a)
	if index < length {
		for i := length; i > index; i-- {
			(*l)[i] = (*l)[i-1]
		}
		(*l)[index] = a
	}
}

func update(units []provision.Unit) {
	log.Print("updating status from provisioner")
	var l AppList
	for _, unit := range units {
		a, index := l.Search(unit.AppName)
		if index > -1 {
			err := a.Get()
			if err != nil {
				log.Printf("collector: app %q not found. Skipping.\n", unit.AppName)
				continue
			}
		}
		u := app.Unit{}
		u.Name = unit.Name
		u.Type = unit.Type
		u.Machine = unit.Machine
		u.InstanceId = unit.InstanceId
		u.Ip = unit.Ip
		u.State = string(unit.Status)
		a.State = string(unit.Status)
		a.Ip = unit.Ip
		a.AddUnit(&u)
		if index > -1 {
			l.Add(a, index)
		}
	}
	for _, a := range l {
		db.Session.Apps().Update(bson.M{"name": a.Name}, a)
	}
}
