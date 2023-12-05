package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

var count int
var report []map[string]interface{}
// ServerDisconnectedError описывает ошибку отключения сервера.
type ServerDisconnectedError struct {
	message string
}

func (e *ServerDisconnectedError) Error() string {
	return e.message
}

// ReportElement определяет интерфейс базового элемента отчета.
type ReportElement interface {
	Init()
	AddToReport(PID int, currStats map[string]string) int
	CreateReport(detailsOrder []string) []map[string]interface{}
}

// ParentElement представляет собой класс родительского элемента.
type ParentElement struct {
	Dimension string
}

func (pe *ParentElement) Init() {}

func (pe *ParentElement) AddToReport(PID int, currStats map[string]string) int {
	myStat := currStats[pe.Dimension]
	for _, i := range report {
		if val, ok := i[pe.Dimension].(string); ok && val == myStat {
			i["Count"] = i["Count"].(int) + 1
			return i["Id"].(int)
		}
	}
	leng := len(report)
	fmt.Println("Len reportov Parent: ", leng)
	newElement := map[string]interface{}{
		"Id":           len(report) + 1, // Adjust the initialization of Id
		"Pid":          nil,
		"URL":          nil,
		"SourceIP":     nil,
		"TimeInterval": nil,
		"Count":        1,
	}
	newElement[pe.Dimension] = myStat
	report = append(report, newElement)
	return newElement["Id"].(int)
}

func (pe *ParentElement) CreateReport(detailsOrder []string) []map[string]interface{} {
	return report
}

// ChildrenElement представляет собой класс дочернего элемента.
type ChildrenElement struct {
	Dimension string
}

func (ce *ChildrenElement) Init() {}

func (ce *ChildrenElement) AddToReport(PID int, currStats map[string]string) int {
	myStat := currStats[ce.Dimension]
	for _, i := range report {
		if val, ok := i[ce.Dimension].(string); ok && val == myStat && i["Pid"].(int) == PID {
			i["Count"] = i["Count"].(int) + 1
			return i["Id"].(int)
		}
	}
	leng := len(report)
	fmt.Println("Len reportov child: ", leng)
	newElement := map[string]interface{}{
		"Id":           len(report) + 1, // Adjust the initialization of Id
		"Pid":          PID,
		"URL":          nil,
		"SourceIP":     nil,
		"TimeInterval": nil,
		"Count":        1,
	}
	newElement[ce.Dimension] = myStat
	report = append(report, newElement)
	return newElement["Id"].(int)
}

func (ce *ChildrenElement) CreateReport(detailsOrder []string) []map[string]interface{} {
	return report
}

// CreatorForJSON представляет собой класс создателя отчетов в формате JSON.
type CreatorForJSON struct {
	Report         []map[string]interface{}
	ReportElements []ReportElement
}

func NewCreatorForJSON(dimension string) *CreatorForJSON {
	fmt.Println("Мы в NewCreatorForJSON:")
	creator := &CreatorForJSON{
		Report:         []map[string]interface{}{},
		ReportElements: []ReportElement{&ParentElement{Dimension: dimension}},
	}
	creator.Init()
	return creator
}

func (c *CreatorForJSON) Init() {
	for _, element := range c.ReportElements {
		element.Init()
	}
}

func (c *CreatorForJSON) AddToReport(PID int, currStats map[string]string) int {
	fmt.Println("Мы в AddToReport:")
	for _, element := range c.ReportElements {
		PID = element.AddToReport(PID, currStats)
	}

	return PID
}

func (c *CreatorForJSON) CreateReport(detailsOrder []string) []map[string]interface{} {
	fmt.Println("Мы в CreateReport:")
	countStr := askDBcount()
	count, err := strconv.Atoi(countStr)
	if err != nil {
		fmt.Println("Error converting counter to int:", err)
		return nil
	}

	c.Report = nil

	for i := 0; i <= count; i++ {
		data := askDB(strconv.Itoa(i))
		stats := map[string]string{"URL": data[0], "SourceIP": data[1], "TimeInterval": data[2]}
		PID := -1
		PID = c.AddToReport(PID, stats)
		fmt.Println("pid у нас: ", PID)
	}

	var report []map[string]interface{}
	for _, element := range c.ReportElements {
		report = append(report, element.CreateReport(detailsOrder)...)
	}

	return report
}

func askDBcount() string {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		handleConnectionError(err)
		return ""
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	_, err = writer.WriteString("HGETSC" + "\n")
	if err != nil {
		handleWriteError(err)
		return ""
	}
	writer.Flush()

	counterStr, err := reader.ReadString('\n')
	if err != nil {
		handleReadError(err)
		return ""
	}
	counterStr = strings.TrimSuffix(counterStr, "\n")

	if counterStr == "" {
		fmt.Println("Empty counter string received")
		return ""
	}
	return counterStr
}

func askDB(istr string) []string {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		handleConnectionError(err)
		return nil
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	_, err = writer.WriteString("HGETS " + istr + "\n")
	if err != nil {
		handleWriteError(err)
		return nil
	}
	writer.Flush()

	dimensions, err := reader.ReadString('\n')
	if err != nil {
		handleReadError(err)
		return nil
	}

	dimensions = strings.TrimSpace(dimensions)
	dimensionsList := strings.Split(dimensions, " ")

	return dimensionsList
}

func handleConnectionError(err error) {
	if strings.Contains(err.Error(), "dial tcp [::1]:6379: connectex: No connection could be made because the target machine actively refused it.") {
		err = &ServerDisconnectedError{"Server disconnected, try again later!"}
		fmt.Println("Server disconnected, try again later!")
	} else {
		fmt.Println("Error connecting:", err)
	}
}

func handleWriteError(err error) {
	fmt.Println("Error writing to server:", err)
}

func handleReadError(err error) {
	fmt.Println("Error reading from server:", err)
}

func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	countStr := strconv.Itoa(count)
	if count == 0 {
		dimensions := strings.Join([]string{countStr, string(body)}, " ")
		resp, err := http.Post("http://localhost:8082/post", "text/plain", strings.NewReader(dimensions))
		if err != nil {
			fmt.Println("Error sending POST request to the second server:", err)
			return
		}
		count++
		defer resp.Body.Close()
	} else {
		dimensions := strings.Join([]string{countStr, string(body)}, " ")
		resp, err := http.Post("http://localhost:8082/post", "text/plain", strings.NewReader(dimensions))
		if err != nil {
			fmt.Println("Error sending POST request to the second server:", err)
			return
		}
		count++
		defer resp.Body.Close()
	}

	fmt.Println("Received data:", string(body))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("POST request received successfully"))
}

func backStatistic(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ya tut")

	decoder := json.NewDecoder(r.Body)
	var data map[string]interface{}
	err := decoder.Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	dimensionsArray, ok := data["Dimensions"].([]interface{})
	if !ok {
		http.Error(w, "Invalid Dimensions format", http.StatusBadRequest)
		return
	}

	dimensions := make([]string, len(dimensionsArray))
	for i, v := range dimensionsArray {
		s, ok := v.(string)
		if !ok {
			http.Error(w, "Invalid Dimension format", http.StatusBadRequest)
			return
		}
		dimensions[i] = s
		fmt.Println("dimensions[i] = s: ", dimensions[i])
	}

	creator := NewCreatorForJSON(dimensions[0])

	for _, dim := range dimensions[1:] {
		child := &ChildrenElement{Dimension: dim}
		var currentElement ReportElement = child
		creator.ReportElements = append(creator.ReportElements, currentElement)
	}

	var detailsOrder []string
	if details, ok := data["Details"].([]interface{}); ok {
		for _, detail := range details {
			if d, ok := detail.(string); ok {
				detailsOrder = append(detailsOrder, d)
			}
		}
	}

	report := creator.CreateReport(detailsOrder)
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}
	w.Write(reportJSON)
}

func main() {
	count = 0
	http.HandleFunc("/post", handlePostRequest)
	http.HandleFunc("/report", backStatistic)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
