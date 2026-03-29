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
	"log/slog"

	"github.com/godbus/dbus/v5"
)

const (
	// NotificationsInterface - freedesktop notifications interface
	NotificationsInterface = "org.freedesktop.Notifications"
	// NotificationsPath - object path для уведомлений
	NotificationsPath = "/org/freedesktop/Notifications"

	// Urgency levels
	UrgencyLow      int32 = 0
	UrgencyNormal   int32 = 1
	UrgencyCritical int32 = 2
)

// Notifier отправляет системные уведомления
type Notifier struct {
	conn *dbus.Conn
}

// NewNotifier создает новый Notifier
func NewNotifier(conn *dbus.Conn) *Notifier {
	return &Notifier{conn: conn}
}

// Notify отправляет уведомление
func (n *Notifier) Notify(title, body string, urgency int32) uint32 {
	if n.conn == nil {
		return 0
	}

	// Проверяем доступность сервиса уведомлений
	if !n.isNotificationsAvailable() {
		slog.Debug("Notifications service not available")
		return 0
	}

	// Параметры уведомления
	appName := "ALR"
	replacesID := uint32(0)
	appIcon := "package-x-generic"
	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(urgency),
	}
	expireTimeout := int32(5000) // 5 секунд

	// Отправляем уведомление
	obj := n.conn.Object(NotificationsInterface, NotificationsPath)
	call := obj.Call(NotificationsInterface+".Notify", 0,
		appName,
		replacesID,
		appIcon,
		title,
		body,
		[]string{},
		hints,
		expireTimeout,
	)

	if call.Err != nil {
		slog.Debug("Failed to send notification", "err", call.Err)
		return 0
	}

	var id uint32
	if err := call.Store(&id); err != nil {
		slog.Debug("Failed to get notification ID", "err", err)
		return 0
	}

	return id
}

// CloseNotification закрывает уведомление
func (n *Notifier) CloseNotification(id uint32) {
	if n.conn == nil || id == 0 {
		return
	}

	obj := n.conn.Object(NotificationsInterface, NotificationsPath)
	call := obj.Call(NotificationsInterface+".CloseNotification", 0, id)

	if call.Err != nil {
		slog.Debug("Failed to close notification", "id", id, "err", call.Err)
	}
}

// isNotificationsAvailable проверяет доступность сервиса уведомлений
func (n *Notifier) isNotificationsAvailable() bool {
	if n.conn == nil {
		return false
	}

	// Проверяем наличие сервиса
	var names []string
	if err := n.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return false
	}

	for _, name := range names {
		if name == NotificationsInterface {
			return true
		}
	}

	return false
}

// NotifyInstallStart уведомление о начале установки
func (n *Notifier) NotifyInstallStart(pkgName string) uint32 {
	return n.Notify(
		"Installing Package",
		"Starting installation of "+pkgName,
		UrgencyNormal,
	)
}

// NotifyInstallComplete уведомление о завершении установки
func (n *Notifier) NotifyInstallComplete(pkgName string) uint32 {
	return n.Notify(
		"Installation Complete",
		pkgName+" has been installed successfully",
		UrgencyNormal,
	)
}

// NotifyInstallError уведомление об ошибке установки
func (n *Notifier) NotifyInstallError(pkgName, errMsg string) uint32 {
	return n.Notify(
		"Installation Failed",
		"Failed to install "+pkgName+": "+errMsg,
		UrgencyCritical,
	)
}

// NotifyUpdateAvailable уведомление о доступных обновлениях
func (n *Notifier) NotifyUpdateAvailable(count int) uint32 {
	return n.Notify(
		"ALR Updates Available",
		"There are "+string(rune('0'+count))+" package updates available",
		UrgencyNormal,
	)
}
