package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	loginCaptchaTTL       = 5 * time.Minute
	loginCaptchaCodeLen   = 4
	loginCaptchaKeyPrefix = "login:captcha:"
)

func (*User) CreateLoginCaptcha(ctx context.Context) (*model.LoginCaptchaRsp, error) {
	if global.REDIS == nil {
		return nil, errcode.New(errcode.CodeSystemError)
	}

	captchaID := uuid.NewString()
	captchaCode, err := generateCaptchaCode(loginCaptchaCodeLen)
	if err != nil {
		return nil, errcode.New(errcode.CodeSystemError)
	}

	if err = global.REDIS.Set(ctx, buildLoginCaptchaKey(captchaID), strings.ToUpper(captchaCode), loginCaptchaTTL).Err(); err != nil {
		return nil, errcode.New(errcode.CodeCacheError)
	}

	svg := buildCaptchaSVG(captchaCode)
	imageDataURI := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))

	return &model.LoginCaptchaRsp{
		CaptchaID:    captchaID,
		CaptchaImage: imageDataURI,
		ExpiresIn:    int64(loginCaptchaTTL.Seconds()),
	}, nil
}

func (*User) VerifyAndConsumeLoginCaptcha(ctx context.Context, captchaID, captchaCode string) error {
	if global.REDIS == nil {
		return errcode.New(errcode.CodeSystemError)
	}

	cacheKey := buildLoginCaptchaKey(captchaID)
	expectedCode, err := global.REDIS.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return errcode.New(200011) // 验证码已过期
		}
		return errcode.New(errcode.CodeCacheError)
	}

	// 验证码为一次性使用，读取后立即销毁。
	_ = global.REDIS.Del(ctx, cacheKey).Err()

	if !strings.EqualFold(strings.TrimSpace(captchaCode), strings.TrimSpace(expectedCode)) {
		return errcode.New(200012) // 验证码错误
	}

	return nil
}

func buildLoginCaptchaKey(captchaID string) string {
	return loginCaptchaKeyPrefix + captchaID
}

func generateCaptchaCode(length int) (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

	var builder strings.Builder
	builder.Grow(length)

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		builder.WriteByte(chars[idx.Int64()])
	}

	return builder.String(), nil
}

func buildCaptchaSVG(code string) string {
	const width = 132
	const height = 44

	line1Y := randomIntInRange(5, height-5)
	line2Y := randomIntInRange(5, height-5)
	line3Y := randomIntInRange(5, height-5)

	char1Y := randomIntInRange(28, 36)
	char2Y := randomIntInRange(28, 36)
	char3Y := randomIntInRange(28, 36)
	char4Y := randomIntInRange(28, 36)

	char1R := randomIntInRange(-15, 15)
	char2R := randomIntInRange(-15, 15)
	char3R := randomIntInRange(-15, 15)
	char4R := randomIntInRange(-15, 15)

	chars := []byte(code)
	for len(chars) < 4 {
		chars = append(chars, 'X')
	}

	return fmt.Sprintf(
		"<svg xmlns='http://www.w3.org/2000/svg' width='%d' height='%d' viewBox='0 0 %d %d'>"+
			"<rect width='100%%' height='100%%' fill='#f8fafc' rx='6' ry='6'/>"+
			"<line x1='8' y1='%d' x2='124' y2='%d' stroke='#cbd5e1' stroke-width='1.2'/>"+
			"<line x1='6' y1='%d' x2='120' y2='%d' stroke='#d1d5db' stroke-width='1.1'/>"+
			"<line x1='10' y1='%d' x2='126' y2='%d' stroke='#e2e8f0' stroke-width='1'/>"+
			"<g font-family='Arial, Helvetica, sans-serif' font-size='24' font-weight='700' fill='#1f2937'>"+
			"<text x='16' y='%d' transform='rotate(%d 16 %d)'>%c</text>"+
			"<text x='44' y='%d' transform='rotate(%d 44 %d)'>%c</text>"+
			"<text x='72' y='%d' transform='rotate(%d 72 %d)'>%c</text>"+
			"<text x='100' y='%d' transform='rotate(%d 100 %d)'>%c</text>"+
			"</g>"+
			"</svg>",
		width, height, width, height,
		line1Y, height-line1Y,
		line2Y, height-line2Y,
		line3Y, height-line3Y,
		char1Y, char1R, char1Y, chars[0],
		char2Y, char2R, char2Y, chars[1],
		char3Y, char3R, char3Y, chars[2],
		char4Y, char4R, char4Y, chars[3],
	)
}

func randomIntInRange(min, max int) int {
	if min >= max {
		return min
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return min
	}

	return min + int(n.Int64())
}
