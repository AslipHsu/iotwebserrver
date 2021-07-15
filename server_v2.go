package main

import (
	// "crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"math"
	"net/http"
	// "net/smtp"
	"strconv"
	"strings"
)

//database login info

const (
	host     = "localhost"
	database = "aiDB"
	user     = "Calvin_Huang"
	// user     = "root"
	password = "123456"
)

// json binding struct
type ElectMsgJson struct {
	//API資料格式
	Customer_user_no     int     `json:"customer_user_no"`
	Gateway_no           int     `json:"gateway_no"`
	Powerprobe_no        int     `json:"powerprobe_no"`
	Data                 float64 `json:"data"`
	Data_sum             float64 `json:"data_sum"`
	VOL                  float64 `json:"VOL"`
	AMP                  float64 `json:"AMP"`
	KW                   float64 `json:"KW"`
	PF                   float64 `json:"PF"`
	ValueDAP             float64 `json:"ValueDAP"`
	ValueDAN             float64 `json:"ValueDAN"`
	ValueDRP             float64 `json:"ValueDRP"`
	ValueDRN             float64 `json:"ValueDRN"`
	ValueAAP             float64 `json:"ValueAAP"`
	ValueAAN             float64 `json:"ValueAAN"`
	ValueARP             float64 `json:"ValueARP"`
	ValueARN             float64 `json:"ValueARN"`
	Year                 int     `json:"year"`
	Month                int     `json:"month"`
	Day                  int     `json:"day"`
	Hour                 int     `json:"hour"`
	Index_date           string  `json:"index_date"`
	Last_update_time     string  `json:"last_update_time"`
	Is_estimate          int     `json:"is_estimate"`
	Ai_power_upperbound1 float64 `json:"ai_power_upperbound1"`
	Ai_power_upperbound2 float64 `json:"ai_power_upperbound2"`
	Ai_power_lowerbound1 float64 `json:"ai_power_lowerbound1"`
	Ai_power_lowerbound2 float64 `json:"ai_power_lowerbound2"`
	Alarm_level_power    int     `json:"alarm_level_power"`
	Alarm_level_voltage  int     `json:"alarm_level_voltage"`
	Alarm_level_ampere   int     `json:"alarm_level_ampere"`
}

type ThMsgJson struct {
	Customer_user_no int     `json:"customer_user_no"`
	Gateway_no       int     `json:"gateway_no"`
	Sensor_no        int     `json:"sensor_no"`
	Sensor_type      int     `json:"sensor_type"`
	Value            float64 `json:"value"`
	Analog_offset    float32 `json:"analog_offset"`
	Year             int     `json:"year"`
	Month            int     `json:"month"`
	Day              int     `json:"day"`
	Hour             int     `json:"hour"`
	Index_date       string  `json:"index_date"`
	Last_update_time string  `json:"last_update_time"`
	Ai_upperbound    float64 `json:"ai_upperbound"`
	Ai_lowerbound    float64 `json:"ai_lowerbound"`
	Alarm_level      int     `json:"alarm_level"`
}

// //send mail
// type mail struct {
// 	user   string
// 	passwd string
// }

//newest record
type probeRecord struct {
	customer_user_no int
	probe            int
	data_sum         float64
	vol              float64
	amp              float64
	index_date       string
	last_update_time string
}

type sensorRecord struct {
	customer_user_no int
	gateway          int
	sensor           int
	index_date       string
	last_update_time string
}

// buttonState
type buttonState struct {
	bProbe  int
	bSensor int
}

var (
	//sse channel
	electMessageChan chan []byte
	thMessageChan    chan []byte
	//newest record
	probeNewRecord  map[int]probeRecord
	sensorNewRecord map[int]sensorRecord
	//buttonState
	bState buttonState
	//error test
	probeErr  int = 0
	sensorErr int = 0
)

func checkErr(err error) {
	if err != nil {
		println(err.Error())
	}
}

// //send mail
// func New(u string, p string) mail {
// 	temp := mail{user: u, passwd: p}
// 	return temp
// }

// func (m mail) Send(title string, text string, toId string) {
// 	auth := smtp.PlainAuth("", m.user, m.passwd, "smtp.gmail.com")
// 	tlsconfig := &tls.Config{
// 		InsecureSkipVerify: true,
// 		ServerName:         "smtp.gmail.com",
// 	}
// 	conn, err := tls.Dial("tcp", "smtp.gmail.com:465", tlsconfig)
// 	checkErr(err)

// 	client, err := smtp.NewClient(conn, "smtp.gmail.com")
// 	checkErr(err)
// 	if err = client.Auth(auth); err != nil {
// 		log.Panic(err)
// 	}
// 	if err = client.Mail(m.user); err != nil {
// 		log.Panic(err)
// 	}
// 	if err = client.Rcpt(toId); err != nil {
// 		log.Panic(err)
// 	}
// 	w, err := client.Data()
// 	checkErr(err)
// 	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", m.user, toId, title, text)
// 	_, err = w.Write([]byte(msg))
// 	checkErr(err)
// 	err = w.Close()
// 	checkErr(err)
// 	client.Quit()
// }

// send line message
func notifyHandler(msg string) {
	authorization_code := `LI8tw1WCkVD63d5MeGnFJSA0cht3rXO7BiaBtX7MmKI`
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://notify-api.line.me/api/notify", strings.NewReader("message="+msg))
	checkErr(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+authorization_code)
	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()
}

func sendProbRecord(probe int) interface{} {
	probeRec := probeNewRecord[probe]
	//conect db
	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowNativePasswords=true", user, password, host, database)
	db, err := sql.Open("mysql", connectionString)
	checkErr(err)

	//take Table_ProbeAlarmThreshold data
	thresholdRows, err := db.Query("SELECT * FROM Table_ProbeAlarmThreshold WHERE powerprobe_no=?", probeRec.probe)
	checkErr(err)
	var powerprobe_no, voltage_upperbound, voltage_lowerbound, ampere_upperbound float64
	for thresholdRows.Next() {
		err = thresholdRows.Scan(&powerprobe_no, &voltage_upperbound, &voltage_lowerbound, &ampere_upperbound)
		checkErr(err)
	}

	//take 1440
	rows, err := db.Query("SELECT * FROM rawDataProbeCSV WHERE index_date <=? and powerprobe_no=? ORDER BY index_date desc limit 1440", probeRec.index_date, probeRec.probe)
	checkErr(err)
	var outputData [1440]interface{}
	count := 0
	fmt.Println("--start package--")
	for rows.Next() {
		var (
			customer_user_no     int
			gateway_no           int
			powerprobe_no        int
			data                 float64
			data_sum             float64
			VOL                  float64
			AMP                  float64
			KW                   float64
			PF                   float64
			ValueDAP             float64
			ValueDAN             float64
			ValueDRP             float64
			ValueDRN             float64
			ValueAAP             float64
			ValueAAN             float64
			ValueARP             float64
			ValueARN             float64
			year                 int
			month                int
			day                  int
			hour                 int
			index_date           string
			last_update_time     string
			is_estimate          int
			ai_power_upperbound1 sql.NullFloat64
			ai_power_upperbound2 sql.NullFloat64
			ai_power_lowerbound1 sql.NullFloat64
			ai_power_lowerbound2 sql.NullFloat64
			alarm_level_power    sql.NullInt32
			alarm_level_voltage  sql.NullInt32
			alarm_level_ampere   sql.NullInt32
			aiPup1               float64
			aiPup2               float64
			aiPlow1              float64
			aiPlow2              float64
			alarmP               int32
			alarmV               int32
			alarmA               int32
		)
		err = rows.Scan(&customer_user_no, &gateway_no, &powerprobe_no, &data, &data_sum, &VOL, &AMP, &KW,
			&PF, &ValueDAP, &ValueDAN, &ValueDRP, &ValueDRN, &ValueAAP, &ValueAAN, &ValueARP, &ValueARN, &year, &month,
			&day, &hour, &index_date, &last_update_time, &is_estimate, &ai_power_upperbound1, &ai_power_upperbound2,
			&ai_power_lowerbound1, &ai_power_lowerbound2, &alarm_level_power, &alarm_level_voltage, &alarm_level_ampere)
		checkErr(err)
		//package data
		// fmt.Println("ai_power_upperbound1", ai_power_upperbound1)
		// fmt.Println("ai_power_upperbound1.valid,%T", ai_power_upperbound1.Valid, ai_power_upperbound1.Valid)
		// fmt.Println("ai_power_upperbound1.Float64,%T", ai_power_upperbound1.Float64, ai_power_upperbound1.Float64)

		if ai_power_upperbound1.Valid && alarm_level_power.Valid {
			aiPup1 = ai_power_upperbound1.Float64
			aiPup2 = ai_power_upperbound2.Float64
			aiPlow1 = ai_power_lowerbound1.Float64
			aiPlow2 = ai_power_lowerbound2.Float64
			alarmP = alarm_level_power.Int32
			alarmV = alarm_level_voltage.Int32
			alarmA = alarm_level_ampere.Int32
		} else {
			aiPup1 = float64(0)
			aiPup2 = float64(0)
			aiPlow1 = float64(0)
			aiPlow2 = float64(0)
			alarmP = int32(0)
			alarmV = int32(0)
			alarmA = int32(0)
		}
		packData := gin.H{"V": gin.H{"index_date": index_date[11:16],
			"date":                 index_date[0:10],
			"value":                VOL,
			"alarm_level":          alarmV,
			"probe":                powerprobe_no,
			"ai_power_upperbound1": voltage_upperbound,
			"ai_power_lowerbound1": voltage_lowerbound,
			"ai_power_upperbound2": voltage_upperbound + math.Abs(voltage_upperbound*0.1),
			"ai_power_lowerbound2": voltage_lowerbound - math.Abs(voltage_lowerbound*0.1)},
			"C": gin.H{"index_date": index_date[11:16],
				"date":                 index_date[0:10],
				"value":                AMP,
				"alarm_level":          alarmA,
				"probe":                powerprobe_no,
				"ai_power_upperbound1": ampere_upperbound,
				"ai_power_upperbound2": ampere_upperbound + math.Abs(ampere_upperbound*0.1)},
			"P": gin.H{"index_date": index_date[11:16],
				"date":                 index_date[0:10],
				"value":                data_sum,
				"alarm_level":          alarmP,
				"probe":                powerprobe_no,
				"ai_power_upperbound1": aiPup1,
				"ai_power_lowerbound1": aiPlow1,
				"ai_power_upperbound2": aiPup2,
				"ai_power_lowerbound2": aiPlow2}}
		outputData[1439-count] = packData
		// fmt.Println("count", count)
		count = count + 1
	}
	db.Close()
	outPut := outputData[:]
	//if data<1440
	if count < 1440 {
		count = count - 1
		outPut = outputData[1439-count:]
	}
	fmt.Println("--packaged: y--")
	return outPut
}

func sendSensorRecord(sensor int) interface{} {
	sensorRec := sensorNewRecord[sensor]
	//connect db
	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowNativePasswords=true", user, password, host, database)
	db, err := sql.Open("mysql", connectionString)
	checkErr(err)
	// take 1440

	var rows *sql.Rows
	if sensorRec.gateway == 101 {
		rows, err = db.Query("SELECT * FROM rawDataSensorCSV WHERE index_date>? and index_date<=? and sensor_no=? ORDER BY index_date desc limit 1440", "20210401000000", sensorRec.index_date, sensorRec.sensor)
		checkErr(err)
	} else {
		rows, err = db.Query("SELECT * FROM rawDataSensorCSV WHERE index_date<=? and sensor_no=? ORDER BY index_date desc limit 1440", sensorRec.index_date, sensorRec.sensor)
		checkErr(err)
	}
	var outputData [1440]interface{}
	count := 0
	fmt.Println("--start package--")
	for rows.Next() {
		var (
			customer_user_no int
			gateway_no       int
			sensor_no        int
			sensor_type      int
			value            float64
			analog_offset    float32
			year             int
			month            int
			day              int
			hour             int
			index_date       string
			last_update_time string
			ai_upperbound    sql.NullFloat64
			ai_lowerbound    sql.NullFloat64
			alarm_level      sql.NullInt32
			aiSup            float64
			aiSlow           float64
			alarmS           int32
		)
		err = rows.Scan(&customer_user_no, &gateway_no, &sensor_no, &sensor_type,
			&value, &analog_offset, &year, &month, &day, &hour, &index_date, &last_update_time,
			&ai_upperbound, &ai_lowerbound, &alarm_level)
		checkErr(err)
		if ai_upperbound.Valid && alarm_level.Valid {
			aiSup = ai_upperbound.Float64
			aiSlow = ai_lowerbound.Float64
			alarmS = alarm_level.Int32
		} else {
			aiSup = float64(0)
			aiSlow = float64(0)
			alarmS = int32(0)
		}
		packData := gin.H{"index_date": index_date[11:16],
			"date":                 index_date[0:10],
			"value":                value,
			"gateway":              gateway_no,
			"sensor":               sensor_no,
			"type":                 sensor_type,
			"alarm_level":          alarmS,
			"ai_power_upperbound1": aiSup,
			"ai_power_lowerbound1": aiSlow,
			"ai_power_upperbound2": aiSup + math.Abs(aiSup*float64(0.1)),
			"ai_power_lowerbound2": aiSlow - math.Abs(aiSlow*float64(0.1))}
		outputData[1439-count] = packData
		count = count + 1
	}
	db.Close()
	outPut := outputData[:]
	//if data<1440
	if count < 1440 {
		count = count - 1
		outPut = outputData[1439-count:]
	}
	fmt.Println("--packaged: y--")
	return outPut
}

func errTest(ctx *gin.Context) {
	probeErr = 1
	sensorErr = 1
	fmt.Println("probeErr:", probeErr)
	fmt.Println("sensorErr:", sensorErr)
}

func getProbes(ctx *gin.Context) {
	// connect db
	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowNativePasswords=true", user, password, host, database)
	db, err := sql.Open("mysql", connectionString)
	checkErr(err)
	// take probes
	probes, err := db.Query("SELECT DISTINCT powerprobe_no FROM Table_ProbeAlarmThreshold")
	checkErr(err)
	db.Close()
	var probeArr []int
	for probes.Next() {
		var probe_no int
		err = probes.Scan(&probe_no)
		checkErr(err)
		probeArr = append(probeArr, probe_no)
	}
	ctx.JSON(200, probeArr)
}

func getGateways(ctx *gin.Context) {
	// connect db
	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowNativePasswords=true", user, password, host, database)
	db, err := sql.Open("mysql", connectionString)
	checkErr(err)
	// take Gateways
	gateways, err := db.Query("SELECT DISTINCT gateway_no FROM Table_Sensor")
	checkErr(err)
	db.Close()
	var gatewayArr []int
	for gateways.Next() {
		var gateway_no int
		err = gateways.Scan(&gateway_no)
		checkErr(err)
		gatewayArr = append(gatewayArr, gateway_no)
	}
	ctx.JSON(200, gatewayArr)
}

func getSensors(ctx *gin.Context) {
	gate := ctx.Query("gateway_no")
	gno, _ := strconv.Atoi(gate)

	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowNativePasswords=true", user, password, host, database)
	db, err := sql.Open("mysql", connectionString)
	checkErr(err)
	// take Sensors
	sensors, err := db.Query("SELECT DISTINCT sensor_no FROM Table_Sensor WHERE gateway_no=?", gno)
	checkErr(err)
	db.Close()
	var sensorArr []int
	for sensors.Next() {
		var sensor_no int
		err = sensors.Scan(&sensor_no)
		checkErr(err)
		sensorArr = append(sensorArr, sensor_no)
	}
	ctx.JSON(200, sensorArr)
}

func electSSE(ctx *gin.Context) {
	//get probe to decide whether send data
	probe_no := ctx.Query("probe_no")
	bState.bProbe, _ = strconv.Atoi(probe_no)
	w := ctx.Writer
	r := ctx.Request
	log.Printf("Get elec handshake from client")
	// prepare the header
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// instantiate the channel
	electMessageChan = make(chan []byte)
	// close the channel after exit the function
	defer func() {
		if electMessageChan != nil {
			close(electMessageChan)
		}
		electMessageChan = nil
		log.Printf("client connection is closed")
	}()
	// prepare the flusher
	flusher, _ := w.(http.Flusher)
	//send newest recorded prob data
	_, ok := probeNewRecord[bState.bProbe]
	if ok {
		packData := sendProbRecord(bState.bProbe)
		jasonData, _ := json.Marshal(packData)
		if len(jasonData) > 0 {
			fmt.Fprintf(w, "data: %s\n\n", jasonData)
			flusher.Flush()
		}
	}
	// trap the request under loop forever
	for {
		select {
		// message will received here and printed
		case elecmessage := <-electMessageChan:
			if len(elecmessage) > 0 {
				fmt.Fprintf(w, "data: %s\n\n", elecmessage)
				flusher.Flush()
			}
		// connection is closed then defer will be executed
		case <-r.Context().Done():
			return
		}
	}
}

func electAPI(ctx *gin.Context) {
	//ShouldBindJSON綁定struct和傳過來的json數據
	var electmsg ElectMsgJson
	if err := ctx.ShouldBindJSON(&electmsg); err != nil {
		//防止EOF錯誤
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{"error": err.Error()})
		return
	}
	//error test
	if probeErr == 1 {
		electmsg.Data_sum = electmsg.Ai_power_upperbound2 + 2.1
		electmsg.Alarm_level_power = 2
		probeErr = 0
		fmt.Println("Data_sum:", electmsg.Data_sum)
		fmt.Println("Ai_power_upperbound2:", electmsg.Ai_power_upperbound2)
	}
	//send message
	if electmsg.Data_sum > electmsg.Ai_power_upperbound2 {
		fmt.Println("send msg")
		tx := fmt.Sprintln("Probe_no:", electmsg.Powerprobe_no, "功耗:", electmsg.Data_sum, "。高出正常值，請立即前往查看 http://web.wasay.cc/")
		notifyHandler(tx)
		// foo := New("bearlinm8866@gmail.com", "googlegoogle")
		// foo.Send("[系統通知]AI監測電力數據異常", tx, "bear_linm8866@yahoo.com.tw")
		fmt.Println("send complete")
	}
	//nwest record
	record := probeRecord{
		customer_user_no: electmsg.Customer_user_no,
		probe:            electmsg.Powerprobe_no,
		data_sum:         electmsg.Data_sum,
		vol:              electmsg.VOL,
		amp:              electmsg.AMP,
		index_date:       electmsg.Index_date,
		last_update_time: electmsg.Last_update_time,
	}
	probeNewRecord[electmsg.Powerprobe_no] = record

	// connect db
	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowNativePasswords=true", user, password, host, database)
	db, err := sql.Open("mysql", connectionString)
	checkErr(err)
	// update data
	sqlStatement, err := db.Prepare("update rawDataProbeCSV set data_sum=?,ai_power_upperbound1=?,ai_power_upperbound2=?,ai_power_lowerbound1=?,ai_power_lowerbound2=?,alarm_level_power=?,alarm_level_voltage=?,alarm_level_ampere=? where index_date=? and last_update_time=? and customer_user_no=? and gateway_no=? and powerprobe_no=?")
	checkErr(err)
	res, err := sqlStatement.Exec(electmsg.Data_sum, electmsg.Ai_power_upperbound1, electmsg.Ai_power_upperbound2, electmsg.Ai_power_lowerbound1, electmsg.Ai_power_lowerbound2, electmsg.Alarm_level_power, electmsg.Alarm_level_voltage, electmsg.Alarm_level_ampere, electmsg.Index_date, electmsg.Last_update_time, electmsg.Customer_user_no, electmsg.Gateway_no, electmsg.Powerprobe_no)
	checkErr(err)
	rowCount, err := res.RowsAffected()
	checkErr(err)
	fmt.Printf("update %d row(s) of data.\n", rowCount)

	if electMessageChan != nil && bState.bProbe == electmsg.Powerprobe_no {
		//take Table_ProbeAlarmThreshold data
		thresholdRows, err := db.Query("SELECT * FROM Table_ProbeAlarmThreshold WHERE powerprobe_no=?", electmsg.Powerprobe_no)
		checkErr(err)
		var powerprobe_no, voltage_upperbound, voltage_lowerbound, ampere_upperbound float64
		for thresholdRows.Next() {
			err = thresholdRows.Scan(&powerprobe_no, &voltage_upperbound, &voltage_lowerbound, &ampere_upperbound)
			checkErr(err)
		}
		// take 1440
		rows, err := db.Query("SELECT * FROM rawDataProbeCSV WHERE index_date <=? and powerprobe_no=? ORDER BY index_date desc limit 1440", electmsg.Index_date, electmsg.Powerprobe_no)
		checkErr(err)

		var outputData [1440]interface{}
		count := 0
		fmt.Println("start package")
		for rows.Next() {
			var (
				customer_user_no     int
				gateway_no           int
				powerprobe_no        int
				data                 float64
				data_sum             float64
				VOL                  float64
				AMP                  float64
				KW                   float64
				PF                   float64
				ValueDAP             float64
				ValueDAN             float64
				ValueDRP             float64
				ValueDRN             float64
				ValueAAP             float64
				ValueAAN             float64
				ValueARP             float64
				ValueARN             float64
				year                 int
				month                int
				day                  int
				hour                 int
				index_date           string
				last_update_time     string
				is_estimate          int
				ai_power_upperbound1 sql.NullFloat64
				ai_power_upperbound2 sql.NullFloat64
				ai_power_lowerbound1 sql.NullFloat64
				ai_power_lowerbound2 sql.NullFloat64
				alarm_level_power    sql.NullInt32
				alarm_level_voltage  sql.NullInt32
				alarm_level_ampere   sql.NullInt32
				aiPup1               float64
				aiPup2               float64
				aiPlow1              float64
				aiPlow2              float64
				alarmP               int32
				alarmV               int32
				alarmA               int32
			)
			err = rows.Scan(&customer_user_no, &gateway_no, &powerprobe_no, &data, &data_sum, &VOL, &AMP, &KW,
				&PF, &ValueDAP, &ValueDAN, &ValueDRP, &ValueDRN, &ValueAAP, &ValueAAN, &ValueARP, &ValueARN, &year, &month,
				&day, &hour, &index_date, &last_update_time, &is_estimate, &ai_power_upperbound1, &ai_power_upperbound2,
				&ai_power_lowerbound1, &ai_power_lowerbound2, &alarm_level_power, &alarm_level_voltage, &alarm_level_ampere)
			checkErr(err)
			//package data
			// fmt.Println("ai_power_upperbound1", ai_power_upperbound1)
			// fmt.Println("ai_power_upperbound1.valid,%T", ai_power_upperbound1.Valid, ai_power_upperbound1.Valid)
			// fmt.Println("ai_power_upperbound1.Float64,%T", ai_power_upperbound1.Float64, ai_power_upperbound1.Float64)

			if ai_power_upperbound1.Valid && alarm_level_power.Valid {
				aiPup1 = ai_power_upperbound1.Float64
				aiPup2 = ai_power_upperbound2.Float64
				aiPlow1 = ai_power_lowerbound1.Float64
				aiPlow2 = ai_power_lowerbound2.Float64
				alarmP = alarm_level_power.Int32
				alarmV = alarm_level_voltage.Int32
				alarmA = alarm_level_ampere.Int32
			} else {
				aiPup1 = float64(0)
				aiPup2 = float64(0)
				aiPlow1 = float64(0)
				aiPlow2 = float64(0)
				alarmP = int32(0)
				alarmV = int32(0)
				alarmA = int32(0)
			}
			packData := gin.H{"V": gin.H{"index_date": index_date[11:16],
				"date":                 index_date[0:10],
				"value":                VOL,
				"alarm_level":          alarmV,
				"probe":                powerprobe_no,
				"ai_power_upperbound1": voltage_upperbound,
				"ai_power_lowerbound1": voltage_lowerbound,
				"ai_power_upperbound2": voltage_upperbound + math.Abs(voltage_upperbound*0.1),
				"ai_power_lowerbound2": voltage_lowerbound - math.Abs(voltage_lowerbound*0.1)},
				"C": gin.H{"index_date": index_date[11:16],
					"date":                 index_date[0:10],
					"value":                AMP,
					"alarm_level":          alarmA,
					"probe":                powerprobe_no,
					"ai_power_upperbound1": ampere_upperbound,
					"ai_power_upperbound2": ampere_upperbound + math.Abs(ampere_upperbound*0.1)},
				"P": gin.H{"index_date": index_date[11:16],
					"date":                 index_date[0:10],
					"value":                data_sum,
					"alarm_level":          alarmP,
					"probe":                powerprobe_no,
					"ai_power_upperbound1": aiPup1,
					"ai_power_lowerbound1": aiPlow1,
					"ai_power_upperbound2": aiPup2,
					"ai_power_lowerbound2": aiPlow2}}

			outputData[1439-count] = packData
			// fmt.Println("count", count)
			count = count + 1
		}
		db.Close()
		outPut := outputData[:]
		//if data<1440
		if count < 1440 {
			count = count - 1
			outPut = outputData[1439-count:]
		}
		fmt.Println(" packaged: y")
		elecMessage, _ := json.Marshal(outPut)
		//透過channel將資料送到elecSSE(),再推到前端
		fmt.Println(" json: y")
		if len(elecMessage) > 0 {
			fmt.Printf("Test2: %s\n", elecMessage)
			electMessageChan <- elecMessage
		}
	}
	db.Close()
}

func thSSE(ctx *gin.Context) {
	sensor_no := ctx.Query("sensor_no")
	bState.bSensor, _ = strconv.Atoi(sensor_no)
	w := ctx.Writer
	r := ctx.Request
	log.Printf("Get thhandshake from client")
	// prepare the header
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// instantiate the channel
	thMessageChan = make(chan []byte)
	// close the channel after exit the function
	defer func() {
		if thMessageChan != nil {
			close(thMessageChan)
		}
		thMessageChan = nil
		log.Printf("client connection is closed")
	}()
	// prepare the flusher
	flusher, _ := w.(http.Flusher)
	//send newest recorded sensor data
	_, ok := sensorNewRecord[bState.bSensor]
	if ok {
		packData := sendSensorRecord(bState.bSensor)
		jasonData, _ := json.Marshal(packData)
		if len(jasonData) > 0 {
			fmt.Fprintf(w, "data: %s\n\n", jasonData)
			flusher.Flush()
		}
	}
	// trap the request under loop forever
	for {
		select {
		// message will received here and printed
		case thmessage := <-thMessageChan:
			if len(thmessage) > 0 {
				fmt.Fprintf(w, "data: %s\n\n", thmessage)
				flusher.Flush()
			}
		// connection is closed then defer will be executed
		case <-r.Context().Done():
			return
		}
	}
}
func thAPI(ctx *gin.Context) {

	//ShouldBindJSON綁定struct和傳過來的json數據
	var thmsg ThMsgJson
	if err := ctx.ShouldBindJSON(&thmsg); err != nil {
		//防止EOF錯誤
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{"error": err.Error()})
		return
	}
	//error test
	if sensorErr == 1 {
		thmsg.Value = thmsg.Ai_upperbound + math.Abs(thmsg.Ai_upperbound*float64(0.1)) + 2.1
		thmsg.Alarm_level = 2
		sensorErr = 0
		fmt.Println("Value:", thmsg.Value)
		fmt.Println("Ai_upperbound2:", thmsg.Ai_upperbound+math.Abs(thmsg.Ai_upperbound*float64(0.1)))
	}
	//send message
	if thmsg.Value > thmsg.Ai_upperbound+math.Abs(thmsg.Ai_upperbound*float64(0.1)) {
		fmt.Println("send msg")
		tx := fmt.Sprintln("Gateway_no:", thmsg.Gateway_no, "Sensor_no:", thmsg.Sensor_no, "溫濕度:", thmsg.Value, " 。數據高出正常值，請立即前往查看 http://web.wasay.cc/")
		notifyHandler(tx)
		// foo := New("bearlinm8866@gmail.com", "googlegoogle")
		// foo.Send("[系統通知]AI監測溫濕度數據異常", tx, "bear_linm8866@yahoo.com.tw")
		fmt.Println("send complete")
	}
	//nwest record
	record := sensorRecord{
		customer_user_no: thmsg.Customer_user_no,
		gateway:          thmsg.Gateway_no,
		sensor:           thmsg.Sensor_no,
		index_date:       thmsg.Index_date,
		last_update_time: thmsg.Last_update_time,
	}
	sensorNewRecord[thmsg.Sensor_no] = record

	// connect db
	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowNativePasswords=true", user, password, host, database)
	db, err := sql.Open("mysql", connectionString)
	checkErr(err)
	// update data
	sqlStatement, err := db.Prepare("update rawDataSensorCSV set value=?,ai_upperbound=?,ai_lowerbound=?,alarm_level=? where index_date=? and last_update_time=? and customer_user_no=? and gateway_no=? and sensor_no=? and sensor_type=?")
	checkErr(err)
	res, err := sqlStatement.Exec(thmsg.Value, thmsg.Ai_upperbound, thmsg.Ai_lowerbound, thmsg.Alarm_level, thmsg.Index_date, thmsg.Last_update_time, thmsg.Customer_user_no, thmsg.Gateway_no, thmsg.Sensor_no, thmsg.Sensor_type)
	checkErr(err)
	rowCount, err := res.RowsAffected()
	checkErr(err)
	fmt.Printf("update %d row(s) of data.\n", rowCount)

	if thMessageChan != nil && bState.bSensor == thmsg.Sensor_no {
		// take 1440
		var rows *sql.Rows
		if thmsg.Gateway_no == 101 {
			rows, err = db.Query("SELECT * FROM rawDataSensorCSV WHERE index_date>? and index_date<=? and sensor_no=? ORDER BY index_date desc limit 1440", "20210401000000", thmsg.Index_date, thmsg.Sensor_no)
			checkErr(err)
		} else {
			rows, err = db.Query("SELECT * FROM rawDataSensorCSV WHERE index_date<=? and sensor_no=? ORDER BY index_date desc limit 1440", thmsg.Index_date, thmsg.Sensor_no)
			checkErr(err)
		}

		var outputData [1440]interface{}
		count := 0
		fmt.Println("start package")
		for rows.Next() {
			var (
				customer_user_no int
				gateway_no       int
				sensor_no        int
				sensor_type      int
				value            float64
				analog_offset    float32
				year             int
				month            int
				day              int
				hour             int
				index_date       string
				last_update_time string
				ai_upperbound    sql.NullFloat64
				ai_lowerbound    sql.NullFloat64
				alarm_level      sql.NullInt32
				aiSup            float64
				aiSlow           float64
				alarmS           int32
			)
			err = rows.Scan(&customer_user_no, &gateway_no, &sensor_no, &sensor_type,
				&value, &analog_offset, &year, &month, &day, &hour, &index_date, &last_update_time,
				&ai_upperbound, &ai_lowerbound, &alarm_level)
			checkErr(err)
			if ai_upperbound.Valid && alarm_level.Valid {
				aiSup = ai_upperbound.Float64
				aiSlow = ai_lowerbound.Float64
				alarmS = alarm_level.Int32
			} else {
				aiSup = float64(0)
				aiSlow = float64(0)
				alarmS = int32(0)
			}
			packData := gin.H{"index_date": index_date[11:16],
				"date":                 index_date[0:10],
				"value":                value,
				"gateway":              gateway_no,
				"sensor":               sensor_no,
				"type":                 sensor_type,
				"alarm_level":          alarmS,
				"ai_power_upperbound1": aiSup,
				"ai_power_lowerbound1": aiSlow,
				"ai_power_upperbound2": aiSup + math.Abs(aiSup*float64(0.1)),
				"ai_power_lowerbound2": aiSlow - math.Abs(aiSlow*float64(0.1))}
			outputData[1439-count] = packData
			count = count + 1
		}
		db.Close()
		outPut := outputData[:]
		//if data<1440
		if count < 1440 {
			count = count - 1
			outPut = outputData[1439-count:]
		}
		fmt.Println(" packaged: y")
		thMessage, _ := json.Marshal(outPut)
		//透過channel將資料送到elecSSE(),再推到前端
		fmt.Println(" json: y")
		thMessageChan <- thMessage
	}
	db.Close()
}
func getProbeNewRecord(ctx *gin.Context) {
	keys := make([]int, 0, len(probeNewRecord))
	for k := range probeNewRecord {
		keys = append(keys, k)
	}
	ctx.JSON(200, keys)
}
func getSensorNewRecord(ctx *gin.Context) {
	data := make(map[int]int)
	for k := range sensorNewRecord {
		data[k] = sensorNewRecord[k].gateway
	}
	ctx.JSON(200, data)
}

// func testElec(ctx *gin.Context) {
// 	ctx.HTML(200, "elect.html", nil) //指定渲染模板
// }
// func testTeHu(ctx *gin.Context) {
// 	ctx.HTML(200, "tehu.html", nil) //指定渲染模板
// }

func main() {
	probeNewRecord = make(map[int]probeRecord)
	sensorNewRecord = make(map[int]sensorRecord)

	r := gin.Default()
	r.Use(cors.Default())
	//測試用頁面
	// r.LoadHTMLGlob("template/*") //load 渲染模板
	// r.GET("/testElec", testElec)
	// r.GET("/testTeHu", testTeHu)

	//error test
	r.GET("/errTest", errTest)

	//See Newest Record data (for devolopement)
	r.GET("/getProbeNewRecord", getProbeNewRecord)
	r.GET("/getSensorNewRecord", getSensorNewRecord)

	//get Probe Gateway Sensor numbers
	r.GET("/getProbes", getProbes)
	r.GET("/getGateways", getGateways)
	r.GET("/getSensors", getSensors)

	//SSE
	r.GET("/electSSE", electSSE)               //即時傳送到前端
	r.POST("/api/aiPostProbeRecord", electAPI) //接API傳送的資料
	r.GET("/thSSE", thSSE)                     //即時傳送到前端
	r.POST("/api/aiPostSensorRecord", thAPI)   //接API傳送的資料
	r.Run(":7321")

}
