package database

import (
	"reflect"
	"time"

	"gorm.io/gorm"
)

// 创建时间的插件
func CreateTimeCallback(createTimeColumn string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement.Schema != nil {
			now := time.Now()
			field, ok := db.Statement.Schema.FieldsByDBName[createTimeColumn]
			if !ok {
				return
			}
			ctx := db.Statement.Context

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Array, reflect.Slice:
				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					reflectValue := db.Statement.ReflectValue.Index(i)
					if _, isZero := field.ValueOf(ctx, reflectValue); isZero {
						err := field.Set(ctx, reflectValue, &now)
						if err != nil {
							db.Logger.Error(ctx, "set %s column fail, err: %+v", createTimeColumn, err)
							return
						}
					}
				}
			case reflect.Struct:
				reflectValue := db.Statement.ReflectValue
				if _, isZero := field.ValueOf(ctx, reflectValue); isZero {
					err := field.Set(ctx, reflectValue, &now)
					if err != nil {
						db.Logger.Error(ctx, "set %s column fail, err: %+v", createTimeColumn, err)
						return
					}
				}
			}
		}
	}
}

// 更新时间的插件
func UpdateTimeCallback(updateTimeColumn string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement.Schema != nil {
			now := time.Now()
			field, ok := db.Statement.Schema.FieldsByDBName[updateTimeColumn]
			if !ok {
				return
			}
			ctx := db.Statement.Context

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Array, reflect.Slice:
				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					reflectValue := db.Statement.ReflectValue.Index(i)

					if _, isZero := field.ValueOf(ctx, reflectValue); isZero {
						err := field.Set(ctx, reflectValue, &now)
						if err != nil {
							db.Logger.Error(ctx, "set %s column fail, err: %+v", updateTimeColumn, err)
							return
						}
					}
				}
			case reflect.Struct:
				reflectValue := db.Statement.ReflectValue

				if _, isZero := field.ValueOf(ctx, reflectValue); isZero {
					err := field.Set(ctx, reflectValue, &now)
					if err != nil {
						db.Logger.Error(ctx, "set %s column fail, err: %+v", updateTimeColumn, err)
						return
					}
				}
			}
		}
	}
}
