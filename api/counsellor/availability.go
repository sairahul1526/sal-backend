package counsellor

import (
	"net/http"
	CONSTANT "salbackend/constant"
	DB "salbackend/database"
	"strconv"
	"time"

	UTIL "salbackend/util"
)

// AvailabilityGet godoc
// @Tags Counsellor Availability
// @Summary Get counsellor availability hours
// @Router /counsellor/availability [get]
// @Param counsellor_id query string true "Counsellor ID to get availability details"
// @Produce json
// @Success 200
func AvailabilityGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var response = make(map[string]interface{})

	// get counsellor availability hours
	availability, status, ok := DB.SelectProcess("select * from "+CONSTANT.SchedulesTable+" where counsellor_id = ?", r.FormValue("counsellor_id"))
	if !ok {
		UTIL.SetReponse(w, status, "", CONSTANT.ShowDialog, response)
		return
	}

	response["availability"] = availability
	UTIL.SetReponse(w, CONSTANT.StatusCodeOk, "", CONSTANT.ShowDialog, response)
}

// AvailabilityUpdate godoc
// @Tags Counsellor Availability
// @Summary Update counsellor availability hours
// @Router /counsellor/availability [put]
// @Param counsellor_id query string true "Counsellor ID to update availability details"
// @Produce json
// @Success 200
func AvailabilityUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var response = make(map[string]interface{})

	// read request body
	body, ok := UTIL.ReadRequestBodyInListMap(r)
	if !ok {
		UTIL.SetReponse(w, CONSTANT.StatusCodeBadRequest, "", CONSTANT.ShowDialog, response)
		return
	}

	for _, day := range body {
		DB.UpdateSQL(CONSTANT.SchedulesTable, map[string]string{"counsellor_id": r.FormValue("counsellor_id"), "weekday": day["weekday"]}, day)
	}

	// get all dates for counsellor and group by weekday
	datesByWeekdays := map[int][]string{}
	availabileDates, status, ok := DB.SelectProcess("select date from "+CONSTANT.SlotsTable+" where counsellor_id = ?", r.FormValue("counsellor_id"))
	if !ok {
		UTIL.SetReponse(w, status, "", CONSTANT.ShowDialog, response)
		return
	}
	dates := UTIL.ExtractValuesFromArrayMap(availabileDates, "date")
	// grouping by weekday
	for _, date := range dates {
		t, _ := time.Parse(date, "2006-01-02")
		datesByWeekdays[int(t.Weekday())] = append(datesByWeekdays[int(t.Weekday())], date)
	}

	// update weekday availability to respective dates
	// will run for 30 days * 24 hours = 720 times - needs to be optimised
	for _, day := range body { // 7 times
		weekday, _ := strconv.Atoi(day["weekday"])
		for _, date := range datesByWeekdays[weekday] { // respective weekday dates i.e., 4-5 times
			for key, value := range day { // 24 times
				DB.ExecuteSQL("update "+CONSTANT.SlotsTable+" set `"+key+"` = "+value+" where counsellor_id = ? and date = ? and `"+key+"` in ("+CONSTANT.SlotUnavailable+", "+CONSTANT.SlotAvailable+")", r.FormValue("counsellor_id"), date) // dont update already booked slots
			}
		}
	}

	UTIL.SetReponse(w, CONSTANT.StatusCodeOk, "", CONSTANT.ShowDialog, response)
}
