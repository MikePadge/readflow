package postgres

import (
	"database/sql"
	"errors"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/ncarlier/readflow/pkg/model"
)

var deviceColumns = []string{
	"id",
	"user_id",
	"key",
	"subscription",
	"created_at",
}

func mapRowToDevice(row *sql.Row) (*model.Device, error) {
	device := &model.Device{}

	sub := ""

	err := row.Scan(
		&device.ID,
		&device.UserID,
		&device.Key,
		&sub,
		&device.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if err = device.SetSubscription(sub); err != nil {
		return nil, err
	}
	return device, nil
}

// CreateDevice create a device
func (pg *DB) CreateDevice(device model.Device) (*model.Device, error) {
	dev, err := pg.GetDeviceByUserIDAndKey(*device.UserID, device.Key)
	if err != nil || dev != nil {
		return dev, err
	}
	sub, err := device.GetSubscription()
	if err != nil {
		return nil, err
	}
	query, args, _ := pg.psql.Insert(
		"devices",
	).Columns(
		"user_id", "key", "subscription",
	).Values(
		device.UserID,
		device.Key,
		sub,
	).Suffix(
		"RETURNING " + strings.Join(deviceColumns, ","),
	).ToSql()

	row := pg.db.QueryRow(query, args...)
	return mapRowToDevice(row)
}

// GetDeviceByID get a device from the DB
func (pg *DB) GetDeviceByID(id uint) (*model.Device, error) {
	query, args, _ := pg.psql.Select(deviceColumns...).From(
		"devices",
	).Where(
		sq.Eq{"id": id},
	).ToSql()
	row := pg.db.QueryRow(query, args...)
	return mapRowToDevice(row)
}

// GetDeviceByUserIDAndKey get an device from the DB
func (pg *DB) GetDeviceByUserIDAndKey(uid uint, key string) (*model.Device, error) {
	query, args, _ := pg.psql.Select(deviceColumns...).From(
		"devices",
	).Where(
		sq.Eq{"user_id": uid},
	).Where(
		sq.Eq{"key": key},
	).ToSql()

	row := pg.db.QueryRow(query, args...)
	return mapRowToDevice(row)
}

// GetDevicesByUserID returns devices of an user from DB
func (pg *DB) GetDevicesByUserID(uid uint) ([]model.Device, error) {
	query, args, _ := pg.psql.Select(deviceColumns...).From(
		"devices",
	).Where(
		sq.Eq{"user_id": uid},
	).ToSql()
	rows, err := pg.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Device

	for rows.Next() {
		device := model.Device{}
		sub := ""
		err = rows.Scan(
			&device.ID,
			&device.UserID,
			&device.Key,
			&sub,
			&device.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Ignore bad subscriptions
		if err = device.SetSubscription(sub); err == nil {
			result = append(result, device)
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteDevice removes an device from the DB
func (pg *DB) DeleteDevice(device model.Device) error {
	query, args, _ := pg.psql.Delete("devices").Where(
		sq.Eq{"id": device.ID},
	).ToSql()
	result, err := pg.db.Exec(query, args...)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("no device has been removed")
	}

	return nil
}

// DeleteDevices removes devices from the DB
func (pg *DB) DeleteDevices(uid uint, ids []uint) (int64, error) {
	query, args, _ := pg.psql.Delete("devices").Where(
		sq.Eq{"user_id": uid},
	).Where(
		sq.Eq{"id": ids},
	).ToSql()
	result, err := pg.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
