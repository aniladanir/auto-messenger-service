package postgresql

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Initialize initializes the db session and auto migrates given models
func Initialize(connStr string, models []any) (db *gorm.DB, err error) {
	retryTicker := time.NewTicker(time.Second * 2)
	defer retryTicker.Stop()

	// retry connect
	for range 5 {
		db, err = gorm.Open(postgres.Open(connStr), &gorm.Config{})
		if err == nil {
			break
		}
		<-retryTicker.C
	}
	if err != nil {
		return
	}

	err = db.AutoMigrate(models...)

	return
}

func Close(db *gorm.DB) error {
	sqlDb, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDb.Close()
}
