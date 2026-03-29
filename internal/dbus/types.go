// ALR - Any Linux Repository
// Copyright (C) 2025 The ALR Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package dbus

import (
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	// DBusWellKnownName - well-known name для ALR сервиса
	DBusWellKnownName = "ru.alr-pkg.ALR"

	// DBusObjectPath - базовый object path
	DBusObjectPath = "/ru/alr_pkg/ALR"

	// Интерфейсы ALR
	ManagerInterfaceName = "ru.alr-pkg.ALR.Manager"
	PackageInterfaceName = "ru.alr-pkg.ALR.Package"
	JobInterfaceName     = "ru.alr-pkg.ALR.Job"

	// PropertiesInterface - стандартный D-Bus Properties interface
	PropertiesInterface = "org.freedesktop.DBus.Properties"

	// ObjectManagerInterface - стандартный D-Bus ObjectManager interface
	ObjectManagerInterface = "org.freedesktop.DBus.ObjectManager"
)

// JobType тип задачи
type JobType string

const (
	JobTypeInstall JobType = "install"
	JobTypeRemove  JobType = "remove"
	JobTypeBuild   JobType = "build"
	JobTypeUpgrade JobType = "upgrade"
	JobTypeRefresh JobType = "refresh"
)

// JobStatus статус задачи
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// PackageInfo представляет информацию о пакете для D-Bus
type PackageInfo struct {
	Name        string
	Repository  string
	Version     string
	Release     uint32
	Epoch       uint32
	Description string
	Summary     string
	Homepage    string
	Licenses    []string
	Installed   bool
	Size        uint64
}

// RepositoryInfo представляет информацию о репозитории
type RepositoryInfo struct {
	Name    string
	URL     string
	Ref     string
	Mirrors []string
}

// JobInfo представляет информацию о задаче
type JobInfo struct {
	ID           uint32
	Type         string
	Status       string
	Progress     float64
	PackageName  string
	Repository   string
	Message      string
	ErrorMessage string
	CreatedAt    int64
	CompletedAt  int64
}

// ToDBus возвращает представление для D-Bus
func (p PackageInfo) ToDBus() []interface{} {
	return []interface{}{
		p.Name,
		p.Repository,
		p.Version,
		p.Description,
		p.Summary,
		p.Homepage,
		p.Licenses,
		p.Installed,
		p.Size,
	}
}

// PackageInfoFromDBus создает PackageInfo из D-Bus данных
func PackageInfoFromDBus(data []interface{}) PackageInfo {
	if len(data) < 9 {
		return PackageInfo{}
	}
	return PackageInfo{
		Name:        data[0].(string),
		Repository:  data[1].(string),
		Version:     data[2].(string),
		Description: data[3].(string),
		Summary:     data[4].(string),
		Homepage:    data[5].(string),
		Licenses:    data[6].([]string),
		Installed:   data[7].(bool),
		Size:        data[8].(uint64),
	}
}

// RepositoryInfoToDBus конвертирует в D-Bus формат
func (r RepositoryInfo) ToDBus() []interface{} {
	return []interface{}{
		r.Name,
		r.URL,
		r.Ref,
		r.Mirrors,
	}
}

// ToDBusMap возвращает JobInfo как map для D-Bus Properties
func (j JobInfo) ToDBusMap() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"Id":           dbus.MakeVariant(j.ID),
		"Type":         dbus.MakeVariant(j.Type),
		"Status":       dbus.MakeVariant(j.Status),
		"Progress":     dbus.MakeVariant(j.Progress),
		"PackageName":  dbus.MakeVariant(j.PackageName),
		"Repository":   dbus.MakeVariant(j.Repository),
		"Message":      dbus.MakeVariant(j.Message),
		"ErrorMessage": dbus.MakeVariant(j.ErrorMessage),
		"CreatedAt":    dbus.MakeVariant(j.CreatedAt),
		"CompletedAt":  dbus.MakeVariant(j.CompletedAt),
	}
}

// NewJobInfo создает новую JobInfo
func NewJobInfo(id uint32, jobType JobType, pkgName, repo string) JobInfo {
	return JobInfo{
		ID:          id,
		Type:        string(jobType),
		Status:      string(JobStatusPending),
		Progress:    0.0,
		PackageName: pkgName,
		Repository:  repo,
		Message:     "",
		CreatedAt:   time.Now().Unix(),
		CompletedAt: 0,
	}
}

// GetPackageObjectPath возвращает object path для пакета
func GetPackageObjectPath(repo, name string) dbus.ObjectPath {
	return dbus.ObjectPath(DBusObjectPath + "/packages/" + repo + "/" + name)
}

// GetJobObjectPath возвращает object path для задачи
func GetJobObjectPath(id uint32) dbus.ObjectPath {
	return dbus.ObjectPath(DBusObjectPath + "/jobs/" + string(rune(id)))
}

// JobObjectPathToID извлекает ID из object path
func JobObjectPathToID(path dbus.ObjectPath) uint32 {
	// Простая реализация - извлечь число из пути
	// /ru/alr_pkg/ALR/jobs/123 -> 123
	return 0 // TODO: implement
}
