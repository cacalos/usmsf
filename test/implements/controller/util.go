package controller

import (
	"errors"
	"strings"

	"camel.uangel.com/ua5g/usmsf.git/dao"
	"github.com/labstack/echo"
)

func (s *NFServer) checkContentType(ctx echo.Context) error {
	//	accept := ctx.Request().Header.Get("accept") // 이거 필요한 소스야.. 임시로 막음
	contentsType := ctx.Request().Header.Get("Content-Type")

	if contentsType == "" {
		err := errors.New("Missing Parameter : Content-Type")
		return err
	}

	if strings.Compare(contentsType, "application/json") != 0 {
		err := errors.New("Invalid Parameter : Unsupported Content-Type")
		return err
	}

	return nil
}

func (s *NFServer) checkSubscriber(supi string,
	amfSupi string,
	body []byte,
) (bool, error) {

	exist := false

	rval, _ := s.mysqlDao.GetSubInfoByKEY(amfSupi) // mysql에 존재하는 경우
	if rval == 1 {
		exist = true
		rval := s.redisDao.InsSub(amfSupi, body)
		if rval == -1 {
			err := errors.New("Service Not Allowed")
			return exist, err
		}
		s.mysqlDao.Delete(amfSupi)

		mbody := &dao.MariaInfo{IMSI: amfSupi, DATA: body}
		s.mysqlDao.Create(mbody)
		return exist, nil
	}

	loggers.InfoLogger().Comment("Add Subscriber Because Does Not Find Subs Info, supi:%s", supi)
	return exist, nil // mysql에 존재하지 않는 경우
}

func (s *NFServer) InsertSubsCriber(supi string,
	amfSupi string,
	body []byte,
) error {

	rval := s.redisDao.InsSub(amfSupi, body)
	if rval == -1 {
		err := errors.New("Context Not Found")
		return err
	}

	mbody := &dao.MariaInfo{IMSI: amfSupi, DATA: body}
	s.mysqlDao.Create(mbody)

	return nil
}
