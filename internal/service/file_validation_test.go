package service

import "testing"

func TestValidateUploadBizType(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		bizType  string
		wantErr  bool
	}{
		{name: "image webp allowed", fileName: "logo.webp", bizType: "image", wantErr: false},
		{name: "apk allowed for app package", fileName: "android.apk", bizType: "appPackage", wantErr: false},
		{name: "hap allowed for app package", fileName: "harmony.hap", bizType: "appPackage", wantErr: false},
		{name: "app allowed for app package", fileName: "desktop.app", bizType: "appPackage", wantErr: false},
		{name: "wgt allowed", fileName: "release.wgt", bizType: "wgtPackage", wantErr: false},
		{name: "csv rejected for image", fileName: "report.csv", bizType: "image", wantErr: true},
		{name: "apk rejected for image", fileName: "android.apk", bizType: "image", wantErr: true},
		{name: "wgt rejected for app package", fileName: "release.wgt", bizType: "appPackage", wantErr: true},
		{name: "invalid biz type path", fileName: "demo.png", bizType: "foo/bar", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUploadBizType(tt.fileName, tt.bizType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateUploadBizType(%q, %q) error = %v, wantErr %v", tt.fileName, tt.bizType, err, tt.wantErr)
			}
		})
	}
}
