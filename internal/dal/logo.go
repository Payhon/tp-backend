package dal

import (
	"context"
	"fmt"

	"project/internal/model"
	query "project/internal/query"
	"project/pkg/global"

	"github.com/sirupsen/logrus"
)

func UpdateLogo(logoID string, logomap map[string]interface{}) error {
	p := query.Logo
	_, err := query.Logo.Where(p.ID.Eq(logoID)).Updates(logomap)
	if err != nil {
		logrus.Error(err)
	}
	return err
}

func GetLogoList() (int64, interface{}, error) {
	q := query.Logo
	var count int64
	queryBuilder := q.WithContext(context.Background())

	logoList, err := queryBuilder.Select().Find()
	if err != nil {
		logrus.Error(err)
		return count, logoList, err
	}
	count, err = queryBuilder.Count()
	return count, logoList, err
}

func EnsureLogoQrcodeColumns() error {
	if global.DB == nil {
		return fmt.Errorf("db not initialized")
	}

	if !global.DB.Migrator().HasColumn(&model.Logo{}, "wxmp_qrcode") {
		if err := global.DB.Exec("ALTER TABLE public.logo ADD COLUMN IF NOT EXISTS wxmp_qrcode varchar(255) NOT NULL DEFAULT ''").Error; err != nil {
			return err
		}
	}
	if !global.DB.Migrator().HasColumn(&model.Logo{}, "app_download_qrcode") {
		if err := global.DB.Exec("ALTER TABLE public.logo ADD COLUMN IF NOT EXISTS app_download_qrcode varchar(255) NOT NULL DEFAULT ''").Error; err != nil {
			return err
		}
	}

	return nil
}
