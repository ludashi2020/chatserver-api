/*
 * @Author: cloudyi.li
 * @Date: 2023-03-29 12:36:21
 * @LastEditTime: 2023-05-24 11:59:48
 * @LastEditors: cloudyi.li
 * @FilePath: /chatserver-api/internal/handler/v1/user/user.go
 */
package user

import (
	"chatserver-api/internal/consts"
	"chatserver-api/internal/model"
	"chatserver-api/internal/service"
	"chatserver-api/pkg/errors"
	"chatserver-api/pkg/errors/ecode"
	"chatserver-api/pkg/logger"
	"chatserver-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userSrv service.UserService
}

func NewUserHandler(_userSrv service.UserService) *UserHandler {
	return &UserHandler{
		userSrv: _userSrv,
	}
}

func (uh *UserHandler) UserGetAvatar() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := uh.userSrv.UserGetAvatar(ctx)
		if err != nil {
			response.JSON(ctx, errors.WithCode(ecode.NotFoundErr, err.Error()), nil)
			return
		}
		response.JSON(ctx, nil, res)
	}
}

func (uh *UserHandler) UserGetInfo() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.GetInt64(consts.UserID)
		userinfo, err := uh.userSrv.UserGetInfo(ctx, id)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.NotFoundErr, "未找到用户信息"), nil)
		} else {
			response.JSON(ctx, nil, userinfo)
		}
	}
}

func (uh *UserHandler) UserRegister() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserRegisterReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		iscode := uh.userSrv.CaptchaVerify(ctx, req.Captcha)
		if !iscode {
			response.JSON(ctx, errors.WithCode(ecode.CaptchaErr, "验证码错误"), nil)
			return
		}
		res, err := uh.userSrv.UserRegister(ctx, req)
		if err != nil {
			uh.userSrv.UserDelete(ctx)
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "未知错误注册失败"), res)
			logger.Error(err.Error())
			return
		}
		err = uh.userSrv.UserActiveGen(ctx)
		if err != nil {
			uh.userSrv.UserDelete(ctx)
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "未知错误注册失败"), res)
			logger.Error(err.Error())
			return
		}
		uh.userSrv.UserInviteVerify(ctx, req.InviteCode)
		response.JSON(ctx, errors.Wrapf(err, ecode.Success, "注册成功,激活链接已发送到您的邮箱 %s 。", req.Email), res)
	}
}

func (uh *UserHandler) UserLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserLoginReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		iscode := uh.userSrv.CaptchaVerify(ctx, req.Captcha)
		if !iscode {
			response.JSON(ctx, errors.WithCode(ecode.CaptchaErr, "验证码错误"), nil)
			return
		}
		res, err := uh.userSrv.UserLogin(ctx, req.Username, req.Password)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.UserLoginErr, "登录失败；账户或密码错误"), nil)
		} else {
			response.JSON(ctx, errors.Wrap(err, ecode.Success, "登录成功"), res)
		}
	}
}

func (uh *UserHandler) UserLogout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenstr := ctx.GetString(consts.TokenCtx)
		err := uh.userSrv.UserLogout(ctx, tokenstr)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.UserLoginErr, "登出失败"), nil)
		} else {
			response.JSON(ctx, errors.Wrap(err, ecode.Success, "登出成功"), nil)
		}
	}
}

func (uh *UserHandler) UserRefresh() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := uh.userSrv.UserRefresh(ctx)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.UserLoginErr, "Token刷新失败"), nil)
		} else {
			response.JSON(ctx, errors.Wrap(err, ecode.Success, "Token刷新成功"), res)
		}
	}
}

func (uh *UserHandler) UserVerifyEmail() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserVerifyEmailReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		res, err := uh.userSrv.UserVerifyEmail(ctx, req.Email)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "接口调用失败"), nil)
		} else {
			response.JSON(ctx, nil, res)
		}
	}
}

func (uh *UserHandler) UserVerifyUserName() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserVerifyUserNameReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		res, err := uh.userSrv.UserVerifyUserName(ctx, req.Username)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "接口调用失败"), nil)
		} else {
			response.JSON(ctx, nil, res)
		}
	}
}

func (uh *UserHandler) UserUpdateNickName() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserUpdateNickNameReq
		id := ctx.GetInt64(consts.UserID)
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		res, err := uh.userSrv.UserUpdateNickName(ctx, id, req.Nickname)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "接口调用失败"), nil)
		} else {
			response.JSON(ctx, nil, res)
		}
	}
}

func (uh *UserHandler) UserActive() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserActiveReq
		if err := ctx.ShouldBindQuery(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}

		if !uh.userSrv.UserTempCodeVerify(ctx, req.ActiveCode) {
			response.JSON(ctx, errors.WithCode(ecode.ActiveErr, "用户激活失败"), nil)
			return
		}
		if err := uh.userSrv.UserActiveChange(ctx); err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.ActiveErr, "用户激活失败"), nil)
			return
		}
		response.JSON(ctx, nil, nil)
	}
}

func (uh *UserHandler) UserPasswordModify() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserPasswordModifyReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		if !uh.userSrv.UserPasswordVerify(ctx, req.OldPassword) {
			response.JSON(ctx, errors.WithCode(ecode.PasswordErr, "密码错误"), nil)
			return
		}
		if err := uh.userSrv.UserPasswordModify(ctx, req.NewPassword); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.Unknown, "密码更新失败"), nil)
			return
		}
		response.JSON(ctx, nil, nil)
	}
}
func (uh *UserHandler) UserPasswordForget() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserForgetReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		iscode := uh.userSrv.CaptchaVerify(ctx, req.Captcha)
		if !iscode {
			response.JSON(ctx, errors.WithCode(ecode.CaptchaErr, "验证码错误"), nil)
			return
		}
		isemail, err := uh.userSrv.UserVerifyEmail(ctx, req.Email)
		if err != nil || isemail.Isvalid {
			response.JSON(ctx, errors.WithCode(ecode.Unknown, "邮箱不存在"), nil)
			return
		}
		if err := uh.userSrv.UserPasswordForget(ctx); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.Unknown, "密码重置邮件发送失败"), nil)
			return
		}
		response.JSON(ctx, nil, nil)
	}
}

func (uh *UserHandler) UserPasswordReset() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.UserPasswordResetReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		if !uh.userSrv.UserTempCodeVerify(ctx, req.TempCode) {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, "验证码错误"), nil)
			return
		}
		if err := uh.userSrv.UserPasswordModify(ctx, req.NewPassword); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.Unknown, "密码更新失败"), nil)
			return
		}
		response.JSON(ctx, nil, nil)
	}
}

func (uh *UserHandler) UserCDkeyPay() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req model.CdkeyPayReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.JSON(ctx, errors.WithCode(ecode.ValidateErr, err.Error()), nil)
			return
		}
		if err := uh.userSrv.UserCDkeyPay(ctx, req.CodeKey); err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.CdKeyErr, "卡密错误"), nil)
			return
		}
		response.JSON(ctx, nil, nil)
	}
}

func (uh *UserHandler) UserGiftCardListGet() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := uh.userSrv.UserGiftCardListGet(ctx)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "接口调用失败"), nil)

		} else {
			response.JSON(ctx, nil, res)

		}
	}
}

func (uh *UserHandler) UserInviteLinkGet() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := uh.userSrv.UserInviteLinkGet(ctx)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "邀请链接获取失败"), nil)
			return
		}
		response.JSON(ctx, nil, res)
	}
}

func (uh *UserHandler) CaptchaGen() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := uh.userSrv.CaptchaGen(ctx)
		if err != nil {
			response.JSON(ctx, errors.Wrap(err, ecode.Unknown, "验证码获取失败"), nil)
			return
		}
		response.JSON(ctx, nil, res)
	}
}
