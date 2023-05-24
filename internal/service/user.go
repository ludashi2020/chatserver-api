/*
 * @Author: cloudyi.li
 * @Date: 2023-03-29 12:37:13
 * @LastEditTime: 2023-05-24 21:39:23
 * @LastEditors: cloudyi.li
 * @FilePath: /chatserver-api/internal/service/user.go
 */
package service

import (
	"chatserver-api/internal/consts"
	"chatserver-api/internal/dao"
	"chatserver-api/internal/model"
	"chatserver-api/internal/model/entity"
	"chatserver-api/pkg/active"
	"chatserver-api/pkg/avatar"
	"chatserver-api/pkg/cache"
	"chatserver-api/pkg/config"
	"chatserver-api/pkg/jwt"
	"chatserver-api/pkg/logger"
	"chatserver-api/pkg/mail"
	"chatserver-api/pkg/verification"
	"chatserver-api/utils/security"
	"chatserver-api/utils/uuid"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var _ UserService = (*userService)(nil)

type UserService interface {
	UserLogout(ctx context.Context, tokenstr string) error
	UserGetByID(ctx context.Context, uid int64) (user entity.User, err error)
	UserRegister(ctx *gin.Context, req model.UserRegisterReq) (res model.UserRegisterRes, err error)
	UserLogin(ctx context.Context, username, password string) (res model.UserLoginRes, err error)
	UserRefresh(ctx *gin.Context) (res model.UserLoginRes, err error)
	UserGetInfo(ctx context.Context, userId int64) (res model.UserGetInfoRes, err error)
	UserGetAvatar(ctx *gin.Context) (res model.UserAvatarRes, err error)
	UserVerifyEmail(ctx *gin.Context, email string) (res model.UserVerifyEmailRes, err error)
	UserVerifyUserName(ctx context.Context, username string) (res model.UserVerifyUserNameRes, err error)
	UserUpdateNickName(ctx context.Context, userId int64, nickname string) (res model.UserUpdateNickNameRes, err error)
	UserActiveGen(ctx *gin.Context) (err error)
	UserActiveChange(ctx *gin.Context) (err error)
	UserDelete(ctx *gin.Context) error
	UserPasswordVerify(ctx *gin.Context, password string) (Isvalid bool)
	UserPasswordModify(ctx *gin.Context, password string) (err error)
	UserPasswordForget(ctx *gin.Context) (err error)
	UserTempCodeVerify(ctx *gin.Context, tempcode string) (Isvalid bool)
	UserTempCodeGen(ctx *gin.Context) (tempcode string, email string, nickname string, err error)
	UserBalanceChange(ctx *gin.Context, userId int64, amount float64) error
	UserCDkeyPay(ctx *gin.Context, codekey string) error
	UserInviteLinkGet(ctx *gin.Context) (res model.UserInviteLinkRes, err error)
	UserInviteGen(ctx *gin.Context) (string, error)
	UserInviteReward(ctx *gin.Context)
	UserInviteVerify(ctx *gin.Context, code string)
	UserGiftCardListGet(ctx *gin.Context) (res model.GiftCardListRes, err error)
	CaptchaGen(ctx *gin.Context) (res model.CaptchaRes, err error)
	CaptchaVerify(ctx *gin.Context, code string) bool
}

// userService 实现UserService接口
type userService struct {
	ud   dao.UserDao
	kd   dao.CDkeyDao
	iSrv uuid.SnowNode
}

func NewUserService(_ud dao.UserDao, _kd dao.CDkeyDao) *userService {
	return &userService{
		ud:   _ud,
		kd:   _kd,
		iSrv: *uuid.NewNode(3),
	}
}

func (us *userService) UserLogin(ctx context.Context, username, password string) (res model.UserLoginRes, err error) {
	userInfo, err := us.ud.UserGetByName(ctx, username)
	if err != nil {
		logger.Infof("查询用户失败%s", err)
		return res, err
	}
	if userInfo.IsActive != true {
		err = errors.New("用户未激活")
		return res, err
	}

	if !security.ValidatePassword(security.PasswordDecryption(password, consts.CBCKEY), userInfo.Password) {
		err = errors.New("Password Error")
		logger.Infof("密码错误%s", username)
		return res, err
	}
	// if userInfo.ExpiredAt.GetUnixTime() < time.Now().Unix() {
	// 	err = errors.New("用户授权过期")
	// 	return res, err
	// }
	timeOut := time.Duration(config.AppConfig.JwtConfig.JwtTtl) * time.Second
	expireAt := time.Now().Add(timeOut)
	claims := jwt.BuildClaims(expireAt, userInfo.Id)
	token, err := jwt.GenToken(claims, config.AppConfig.JwtConfig.Secret)
	if err != nil {
		logger.Infof("JWTTOKEN生成错误%s", username)

		return res, err
	}
	res.Token = token
	res.TimeOut = int(config.AppConfig.JwtConfig.JwtTtl) * 1000
	return res, err
}

func (us *userService) UserRefresh(ctx *gin.Context) (res model.UserLoginRes, err error) {
	userId := ctx.GetInt64(consts.UserID)
	tokenStr := ctx.GetString(consts.TokenCtx)
	userInfo, err := us.ud.UserGetById(ctx, userId)
	if err != nil {
		logger.Infof("查询用户失败%s", err)
		return res, err
	}
	if userInfo.IsActive != true {
		err = errors.New("用户未激活")
		return res, err
	}
	// if userInfo.ExpiredAt.GetUnixTime() < time.Now().Unix() {
	// 	err = errors.New("用户授权过期")
	// 	return res, err
	// }
	timeOut := time.Duration(config.AppConfig.JwtConfig.JwtTtl) * time.Second
	expireAt := time.Now().Add(timeOut)
	claims := jwt.BuildClaims(expireAt, userInfo.Id)
	token, err := jwt.GenToken(claims, config.AppConfig.JwtConfig.Secret)
	if err != nil {
		logger.Infof("JWTTOKEN生成错误%s", userInfo.Username)

		return res, err
	}
	res.Token = token
	res.TimeOut = int(config.AppConfig.JwtConfig.JwtTtl) * 1000
	err = jwt.JoinBlackList(ctx, tokenStr, config.AppConfig.JwtConfig.Secret)
	if err != nil {
		logger.Infof("加入黑名单失败%s", userInfo.Username)
	}
	return res, err
}

// GetByName 通过用户名 查找用户
func (us *userService) UserGetByID(ctx context.Context, uid int64) (user entity.User, err error) {
	return us.ud.UserGetById(ctx, uid)
}

func (us *userService) UserGetAvatar(ctx *gin.Context) (res model.UserAvatarRes, err error) {
	userId := ctx.GetInt64(consts.UserID)
	rc := cache.GetRedisClient()
	avatar_url, err := rc.Get(ctx, consts.UserAvatarPrefix+strconv.FormatInt(userId, 10)).Result()
	if avatar_url == "" || err != nil {
		avatar_url, err = us.ud.UserGetAvatar(ctx, userId)
		if err != nil {
			return
		}
		err = rc.SetNX(ctx, consts.UserAvatarPrefix+strconv.FormatInt(userId, 10), avatar_url, -1).Err()
		if err != nil {
			return
		}
	}
	data, err := os.ReadFile(avatar_url)
	if err != nil {
		return
	}
	res.Avatar = base64.StdEncoding.EncodeToString(data)
	return
}

func (us *userService) UserGetInfo(ctx context.Context, userId int64) (res model.UserGetInfoRes, err error) {
	user, err := us.ud.UserGetById(ctx, userId)
	if err != nil {
		return res, err
	}
	res.Balance = user.Balance
	res.Email = user.Email
	res.Nickname = user.Nickname
	res.Username = user.Username
	res.Phone = user.Phone
	res.Role = consts.RoleToString[user.Role]
	// res.ExpiredAt = jtime.JsonTime(user.ExpiredAt)
	return res, err
}

func (us *userService) UserRegister(ctx *gin.Context, req model.UserRegisterReq) (res model.UserRegisterRes, err error) {
	user := entity.User{}
	res.IsSuccess = false
	user.Id = us.iSrv.GenSnowID()
	ctx.Set(consts.UserID, user.Id)
	user.Username = req.Username
	user.Nickname = req.Username
	user.RegisteredIp = ctx.ClientIP()
	user.Email = req.Email
	user.Role = consts.StandardUser
	// user.ExpiredAt = jtime.JsonTime(time.Now().AddDate(0, 0, 7))
	user.Balance = 10
	user.IsActive = false
	user.AvatarUrl, err = avatar.GenNewAvatar(security.Md5WithSalt(req.Username, req.Email))
	if err != nil {
		return res, err
	}
	user.Password, err = security.Encrypt(security.PasswordDecryption(req.Password, consts.CBCKEY))
	if err != nil {
		return res, err
	}
	err = us.ud.UserCreate(ctx, &user)
	if err != nil {
		return res, err
	}
	res.IsSuccess = true
	return
}

func (us *userService) UserPasswordVerify(ctx *gin.Context, password string) (Isvalid bool) {
	userId := ctx.GetInt64(consts.UserID)
	Isvalid = false
	userInfo, err := us.ud.UserGetById(ctx, userId)
	if err != nil {
		logger.Infof("查询用户失败%s", err)
		return
	}
	if !security.ValidatePassword(security.PasswordDecryption(password, consts.CBCKEY), userInfo.Password) {
		// err = errors.New("Password Error")
		logger.Infof("密码错误%s", userInfo.Username)
		return
	}
	return true
}

func (us *userService) UserPasswordModify(ctx *gin.Context, password string) (err error) {
	user := entity.User{}
	userId := ctx.GetInt64(consts.UserID)
	user.Id = userId
	user.Password, err = security.Encrypt(security.PasswordDecryption(password, consts.CBCKEY))
	if err != nil {
		return err
	}
	err = us.ud.UserUpdate(ctx, &user)
	return err
}

func (us *userService) UserPasswordForget(ctx *gin.Context) (err error) {
	tempcode, email, nikcname, err := us.UserTempCodeGen(ctx)
	//19+16 35
	err = mail.SendForgetCode(email, nikcname, tempcode)
	return
}

func (us *userService) UserVerifyEmail(ctx *gin.Context, email string) (res model.UserVerifyEmailRes, err error) {
	UserId, err := us.ud.UserVerifyEmail(ctx, email)
	if err != nil {
		return
	}
	if UserId != 0 {
		res.Isvalid = false
		ctx.Set(consts.UserID, UserId)
	} else {
		res.Isvalid = true
	}
	logger.Debugf("邮箱校验信息：%d", UserId)
	return
}

func (us *userService) UserVerifyUserName(ctx context.Context, username string) (res model.UserVerifyUserNameRes, err error) {
	count, err := us.ud.UserVerifyUserName(ctx, username)
	if err != nil {
		return
	}
	if count != 0 {
		res.Isvalid = false
	} else {
		res.Isvalid = true
	}
	return
}

func (us *userService) UserUpdateNickName(ctx context.Context, userId int64, nickname string) (res model.UserUpdateNickNameRes, err error) {
	err = us.ud.UserUpdateNickName(ctx, userId, nickname)
	if err != nil {
		res.IsChanged = false
	} else {
		res.IsChanged = true
	}
	return
}

func (us *userService) UserLogout(ctx context.Context, tokenstr string) error {
	return jwt.JoinBlackList(ctx, tokenstr, config.AppConfig.JwtConfig.Secret)
}

func (us *userService) UserDelete(ctx *gin.Context) error {
	userId := ctx.GetInt64(consts.UserID)
	return us.ud.UserDelete(ctx, userId)
}

func (us *userService) UserTempCodeGen(ctx *gin.Context) (tempcode string, email string, nickname string, err error) {
	userId := ctx.GetInt64(consts.UserID)
	userInfo, err := us.ud.UserGetById(ctx, userId)
	code, err := active.ActiveCodeGen(ctx, userId)
	if err != nil {
		return
	}
	email = userInfo.Email
	nickname = userInfo.Nickname
	tempcode = base64.StdEncoding.EncodeToString([]byte(code + "|" + userInfo.Username))
	return
}

func (us *userService) UserTempCodeVerify(ctx *gin.Context, tempcode string) (Isvalid bool) {
	Isvalid = false
	codeStr, err := base64.StdEncoding.DecodeString(tempcode)
	if err != nil {
		return
	}
	codelist := strings.Split(string(codeStr), "|")
	if len(codelist) < 2 {
		// err = errors.New("Active Failed")
		return
	}
	code := codelist[0]
	username := codelist[1]
	userInfo, err := us.ud.UserGetByName(ctx, username)
	if err != nil {
		return
	}
	ctx.Set(consts.UserID, userInfo.Id)
	active := active.ActiveCodeCompare(ctx, code, userInfo.Id)
	if !active {
		Isvalid = false
	} else {
		Isvalid = true
	}
	return
}

func (us *userService) UserActiveGen(ctx *gin.Context) (err error) {
	tempcode, email, nikcname, err := us.UserTempCodeGen(ctx)
	//19+16 35
	err = mail.SendActiceCode(email, nikcname, tempcode)
	return
}

func (us *userService) UserActiveChange(ctx *gin.Context) (err error) {
	userId := ctx.GetInt64(consts.UserID)
	user := entity.User{}
	user.Id = userId
	user.IsActive = true
	err = us.ud.UserUpdate(ctx, &user)
	if err != nil {
		return err
	}
	us.UserInviteReward(ctx)

	for i := 1; i <= 3; i++ {
		_, err = us.UserInviteGen(ctx)
		if err == nil {
			break
		}
		if i == 3 {
			break
		}
		time.Sleep(1)
	}

	return nil
}

func (us *userService) UserBalanceChange(ctx *gin.Context, userId int64, amount float64) error {
	oldbalance, err := us.ud.UserGetBalance(ctx, userId)
	if err != nil {
		return err
	}
	newbalance := oldbalance + amount
	user := entity.User{}
	user.Id = userId
	user.Balance = newbalance

	err = us.ud.UserUpdate(ctx, &user)
	if err != nil {
		return err
	}
	return nil
}

func (us *userService) UserCDkeyPay(ctx *gin.Context, key string) error {
	userId := ctx.GetInt64(consts.UserID)
	keyId := uuid.CodeToId(key)
	if keyId == 0 {
		return errors.New("CDKEY ERROR")
	}
	keyAmount, err := us.kd.CdKeyQuery(ctx, keyId)
	if err != nil {
		return err
	}
	if keyAmount.CodeKey != key {
		return errors.New("CDKEY ERROR")
	}
	err = us.UserBalanceChange(ctx, userId, keyAmount.CardAmount)
	if err != nil {
		return err
	}
	err = us.kd.CdKeyDelete(ctx, keyId)
	if err != nil {
		logger.Errorf("CDKEY删除异常：%v", err)
	}
	return nil
}
func (us *userService) UserInviteGen(ctx *gin.Context) (string, error) {
	codeId := us.iSrv.GenSnowID()
	userId := ctx.GetInt64(consts.UserID)
	invite := &entity.Invite{}
	invite.Id = codeId
	invite.UserId = userId
	invite.InviteCode = uuid.GetInvCodeByUID(codeId)
	return invite.InviteCode, us.ud.UserInviteGen(ctx, invite)
}

func (us *userService) UserInviteLinkGet(ctx *gin.Context) (res model.UserInviteLinkRes, err error) {
	userId := ctx.GetInt64(consts.UserID)
	invite, err := us.ud.UserInviteGetByUser(ctx, userId)
	if err != nil {
		return
	}
	code := invite.InviteCode
	if code == "" {
		for i := 1; i <= 3; i++ {
			code, err = us.UserInviteGen(ctx)
			if err == nil {
				break
			}
			if i == 3 {
				break
			}
			time.Sleep(1)
		}
	}
	res.InviteLink = config.AppConfig.ExternalURL + "#/register/" + code
	res.InviteNumber = invite.InviteNumber
	res.InviteReward = float64(invite.InviteNumber * consts.InviteReward)
	return
}

func (us *userService) UserInviteReward(ctx *gin.Context) {
	current_userId := ctx.GetInt64(consts.UserID)
	rc := cache.GetRedisClient()
	invite_str, err := rc.Get(ctx, consts.UserInvitePrefix+strconv.FormatInt(current_userId, 10)).Result()
	if invite_str == "" {
		// logger.Errorf("UserID：%v",)
		return
	}
	invite_userId, err := strconv.ParseInt(invite_str, 10, 64)
	err = us.UserBalanceChange(ctx, invite_userId, consts.InviteReward)
	if err != nil {
		logger.Errorf("UserID：%v 获取奖励失败", invite_userId)
		return
	}
	err = us.UserBalanceChange(ctx, current_userId, consts.InviteReward)
	if err != nil {
		logger.Errorf("current_userId:%v 获取奖励失败", current_userId)
		return
	}
	invite, err := us.ud.UserInviteGetByUser(ctx, invite_userId)
	if err != nil {
		logger.Errorf("UserID：%v 获取Invite信息失败", invite_userId)
		return
	}
	invite.InviteNumber += 1
	err = us.ud.UserInviteUpdate(ctx, &entity.Invite{Id: invite.Id, InviteNumber: invite.InviteNumber})
	if err != nil {
		logger.Errorf("UserID：%v 更新邀请次数失败", invite_userId)
		return
	}
	return
}

func (us *userService) UserInviteVerify(ctx *gin.Context, code string) {
	current_userId := ctx.GetInt64(consts.UserID)
	invite, err := us.ud.UserInviteGetByCode(ctx, code)
	if err != nil {
		logger.Errorf("current_userId:%v 邀请码错误", current_userId)
		return
	}
	invite_userId := invite.UserId
	rc := cache.GetRedisClient()
	timer := 172800 * time.Second
	err = rc.SetNX(ctx, consts.UserInvitePrefix+strconv.FormatInt(current_userId, 10), invite_userId, timer).Err()
	if err != nil {
		return
	}
	return
}

func (us *userService) UserGiftCardListGet(ctx *gin.Context) (res model.GiftCardListRes, err error) {
	var giftcardOne model.GiftCardOneRes
	var giftcardlistRes []model.GiftCardOneRes
	giftcardlist, err := us.kd.GiftCardListGet(ctx)
	if err != nil {
		return
	}
	for _, v := range giftcardlist {
		giftcardOne.CardId = strconv.FormatInt(v.CardId, 10)
		giftcardOne.CardName = v.CardName
		giftcardOne.CardAmount = v.CardAmount
		giftcardOne.CardComment = v.CardComment
		giftcardOne.CardDiscount = v.CardDiscount
		giftcardOne.CardLink = v.CardLink
		giftcardlistRes = append(giftcardlistRes, giftcardOne)
	}
	res.GiftCardList = giftcardlistRes
	return
}

func (us *userService) CaptchaGen(ctx *gin.Context) (res model.CaptchaRes, err error) {
	image, err := verification.GenerateCaptcha(ctx)
	res.Image = image
	return
}

func (us *userService) CaptchaVerify(ctx *gin.Context, code string) bool {
	return verification.VerifyCaptcha(ctx, code)
}
