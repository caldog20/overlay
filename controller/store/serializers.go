package store

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"reflect"

	"gorm.io/gorm/schema"
)

type AddrSerializer struct{}

func (AddrSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	switch value := dbValue.(type) {
	case []byte:
		err = field.Set(ctx, dst, netip.MustParseAddr(string(value)))
	case string:
		err = field.Set(ctx, dst, netip.MustParseAddr(value))
	default:
		return fmt.Errorf("Error deserializing addr value %#v: %w", dbValue.(string), err)
	}

	return
}

func (AddrSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue.(netip.Addr).IsValid() {
		return fieldValue.(netip.Addr).String(), nil
	}
	return nil, errors.New("error serializing peer IP to database, invalid IP")
}

type AddrPortSerializer struct{}

func (AddrPortSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	switch value := dbValue.(type) {
	case []byte:
		err = field.Set(ctx, dst, netip.MustParseAddrPort(string(value)))
	case string:
		err = field.Set(ctx, dst, netip.MustParseAddrPort(value))
	default:
		return fmt.Errorf("Error deserializing addr/port value %#v: %w", dbValue.(string), err)
	}
	return
}

func (AddrPortSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue.(netip.AddrPort).IsValid() {
		return fieldValue.(netip.AddrPort).String(), nil
	}
	return nil, errors.New("error serializing peer addr/port to database, invalid addr/port")
}
