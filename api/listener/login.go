package listener

import (
	"net/http"
	CONFIG "salbackend/config"
	CONSTANT "salbackend/constant"
	DB "salbackend/database"

	UTIL "salbackend/util"
	"strings"
)

// SendOTP godoc
// @Tags Listener Login
// @Summary Send OTP to specified phone
// @Router /listener/sendotp [get]
// @Param phone query string true "Phone number to send OTP - send phone number with 91 code"
// @Produce json
// @Success 200
func SendOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var response = make(map[string]interface{})

	if len(r.FormValue("phone")) != 10 {
		UTIL.SetReponse(w, CONSTANT.StatusCodeBadRequest, CONSTANT.ValidPhoneRequiredMessage, CONSTANT.ShowDialog, response)
		return
	}

	// get listener details
	listener, status, ok := DB.SelectSQL(CONSTANT.ListenersTable, []string{"status"}, map[string]string{"phone": r.FormValue("phone")})
	if !ok {
		UTIL.SetReponse(w, status, "", CONSTANT.ShowDialog, response)
		return
	}
	if len(listener) > 0 && !strings.EqualFold(listener[0]["status"], CONSTANT.ListenerActive) {
		UTIL.SetReponse(w, CONSTANT.StatusCodeBadRequest, CONSTANT.ListenerAccountDeletedMessage, CONSTANT.ShowDialog, response)
		return
	}

	// using phone verified table to check if phone has been really verified by OTP
	// currently deleting if phone number is already present
	DB.DeleteSQL(CONSTANT.PhoneOTPVerifiedTable, map[string]string{"phone": r.FormValue("phone")})

	// TODO send otp using msg91

	UTIL.SetReponse(w, CONSTANT.StatusCodeOk, "", CONSTANT.ShowDialog, response)
}

// VerifyOTP godoc
// @Tags Listener Login
// @Summary Verify OTP sent to specified phone
// @Router /listener/verifyotp [get]
// @Param phone query string true "Phone number OTP has been sent to - send phone number with 91 code"
// @Param otp query string true "OTP entered by listener"
// @Produce json
// @Success 200
func VerifyOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var response = make(map[string]interface{})

	// TODO check if otp is correct
	if !strings.EqualFold(r.FormValue("otp"), "4242") {
		UTIL.SetReponse(w, CONSTANT.StatusCodeBadRequest, CONSTANT.IncorrectOTPRequiredMessage, CONSTANT.ShowDialog, response)
		return
	}

	// get listener details
	listener, status, ok := DB.SelectSQL(CONSTANT.ListenersTable, []string{"*"}, map[string]string{"phone": r.FormValue("phone")})
	if !ok {
		UTIL.SetReponse(w, status, "", CONSTANT.ShowDialog, response)
		return
	}
	if len(listener) == 0 {
		// listener is first time signing up

		// using phone verified table to check if phone has been really verified by OTP
		// currently inserting after phone is really verified
		DB.InsertSQL(CONSTANT.PhoneOTPVerifiedTable, map[string]string{"phone": r.FormValue("phone"), "created_at": UTIL.GetCurrentTime().String()})
	} else {
		// listener already signed up
		// check if listener is active
		if !strings.EqualFold(listener[0]["status"], CONSTANT.ListenerActive) {
			UTIL.SetReponse(w, CONSTANT.StatusCodeBadRequest, CONSTANT.ListenerAccountDeletedMessage, CONSTANT.ShowDialog, response)
			return
		}

		// generate access and refresh token
		// access token - jwt token with short expiry added in header for authorization
		// refresh token - jwt token with long expiry to get new access token if expired
		// if refresh token expired, need to login
		accessToken, ok := UTIL.CreateAccessToken(listener[0]["listener_id"])
		if !ok {
			UTIL.SetReponse(w, CONSTANT.StatusCodeServerError, "", CONSTANT.ShowDialog, response)
			return
		}
		refreshToken, ok := UTIL.CreateRefreshToken(listener[0]["listener_id"])
		if !ok {
			UTIL.SetReponse(w, CONSTANT.StatusCodeServerError, "", CONSTANT.ShowDialog, response)
			return
		}

		response["access_token"] = accessToken
		response["refresh_token"] = refreshToken

		response["listener"] = listener[0]
		response["media_url"] = CONFIG.MediaURL
	}

	UTIL.SetReponse(w, CONSTANT.StatusCodeOk, "", CONSTANT.ShowDialog, response)
}

// RefreshToken godoc
// @Tags Listener Login
// @Summary Get new access token with refresh token
// @Router /listener/refresh-token [get]
// @Param listener_id query string true "Logged in listener ID"
// @Produce json
// @Success 200
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var response = make(map[string]interface{})

	// check if refresh token is valid, not expired and token user id is same as user id given
	id, ok, access := UTIL.ParseJWTAccessToken(r.Header.Get("Authorization"))
	if !ok || access || !strings.EqualFold(id, r.FormValue("listener_id")) {
		UTIL.SetReponse(w, CONSTANT.StatusCodeSessionExpired, CONSTANT.SessionExpiredMessage, CONSTANT.ShowDialog, response)
		return
	}

	// check if listener id is valid
	if DB.CheckIfExists(CONSTANT.ListenersTable, map[string]string{"listener_id": r.FormValue("listener_id"), "status": CONSTANT.ListenerActive}) {
		UTIL.SetReponse(w, CONSTANT.StatusCodeBadRequest, CONSTANT.ListenerNotExistMessage, CONSTANT.ShowDialog, response)
		return
	}

	// generate new access token
	accessToken, ok := UTIL.CreateAccessToken(r.FormValue("listener_id"))
	if !ok {
		UTIL.SetReponse(w, CONSTANT.StatusCodeSessionExpired, CONSTANT.SessionExpiredMessage, CONSTANT.ShowDialog, response)
		return
	}

	response["access_token"] = accessToken

	UTIL.SetReponse(w, CONSTANT.StatusCodeOk, "", CONSTANT.ShowDialog, response)
}
