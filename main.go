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
	Report    []map[string]interface{}
}
// Init инициализирует родительский элемент перед обработкой данных из базы данных.
func (pe *ParentElement) Init() {
	pe.Report = nil
}
// AddToReport добавляет элемент в родительскую часть отчета.
func (pe *ParentElement) AddToReport(PID int, currStats map[string]string) int {
	fmt.Println("Мы в ce ParentElement) AddToReport ")
	myStat := currStats[pe.Dimension]
	for _, i := range pe.Report {
		if val, ok := i[pe.Dimension].(string); ok && val == myStat {
			i["Count"] = i["Count"].(int) + 1
			return i["Id"].(int)
		}
	}
	newElement := map[string]interface{}{
		"Id":           len(pe.Report)+1,
		"Pid":          nil,
		"URL":          nil,
		"SourceIP":     nil,
		"TimeInterval": nil,
		"Count":        1,
	}
	newElement[pe.Dimension] = myStat
	pe.Report = append(pe.Report, newElement)
	return newElement["Id"].(int)
}

// CreateReport создает отчет в формате JSON.
func (pe *ParentElement) CreateReport(detailsOrder []string) []map[string]interface{} {
	return pe.Report
}

// ChildrenElement представляет собой класс дочернего элемента.
type ChildrenElement struct {
	Dimension string
	Report    []map[string]interface{}
}
// Init инициализирует дочерний элемент перед обработкой данных из базы данных.
func (ce *ChildrenElement) Init() {
	ce.Report = nil
}
// AddToReport добавляет элемент в дочернюю часть отчета.
func (ce *ChildrenElement) AddToReport(PID int, currStats map[string]string) int {
	fmt.Println("Мы в ce *ChildrenElement) AddToReport:")
	myStat := currStats[ce.Dimension]
	for _, i := range ce.Report {
		if val, ok := i[ce.Dimension].(string); ok && val == myStat && i["Pid"].(int) == PID {
			i["Count"] = i["Count"].(int) + 1
			return i["Id"].(int)
		}
	}
	newElement := map[string]interface{}{
		"Id":           len(ce.Report)+1,
		"Pid":          PID,
		"URL":          nil,
		"SourceIP":     nil,
		"TimeInterval": nil,
		"Count":        1,
	}
	newElement[ce.Dimension] = myStat
	ce.Report = append(ce.Report, newElement)
	return newElement["Id"].(int)
}

// CreateReport создает отчет в формате JSON.
func (ce *ChildrenElement) CreateReport(detailsOrder []string) []map[string]interface{} {
	return ce.Report
}

// CreatorForJSON представляет собой класс создателя отчетов в формате JSON.
type CreatorForJSON struct {
	ReportElements []ReportElement
}

// NewCreatorForJSON initializes an instance of CreatorForJSON.
func NewCreatorForJSON(dimension string) *CreatorForJSON {
	fmt.Println("Мы в NewCreatorForJSON:")
	creator := &CreatorForJSON{
		ReportElements: []ReportElement{&ParentElement{Dimension: dimension}},
	}
	creator.Init()
	return creator
}

// Init initializes CreatorForJSON before processing data from the database.
func (c *CreatorForJSON) Init() {
	for _, element := range c.ReportElements {
		element.Init()
	}
}

// AddToReport adds an element to the report using the appropriate method.
func (c *CreatorForJSON) AddToReport(PID int, currStats map[string]string) int {
	fmt.Println("Мы в AddToReport:")
	for _, element := range c.ReportElements {
		PID = element.AddToReport(PID, currStats)
	}
	return PID
}

// CreateReport creates a JSON report.
func (c *CreatorForJSON) CreateReport(detailsOrder []string) []map[string]interface{} {
	fmt.Println("Мы в CreateReport:")
	countStr := askDBcount()
	count, err := strconv.Atoi(countStr)
	if err != nil {
		fmt.Println("Error converting counter to int:", err)
		return nil
	}

	c.Init()

	for i := 0; i <= count; i++ {
		data := askDB(strconv.Itoa(i))
		stats := map[string]string{"URL": data[0], "SourceIP": data[1], "TimeInterval": data[2]}
		PID := -1
		PID = c.AddToReport(PID, stats)
		fmt.Println("pid у нас: ", PID)
	}

	// Assuming that the first element is a ParentElement
	var report []map[string]interface{}
	for _, element := range c.ReportElements {
		report = append(report, element.CreateReport(detailsOrder)...)
	}

	return report
}

// askDBcount - функция для выполнения запросов к базе данных для получения счетчика.
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

// askDB - функция для выполнения запросов к базе данных.
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

	// Разделение измерений по пробелу
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

	// Чтение данных из тела запроса
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

	// Вывод полученных данных
	fmt.Println("Received data:", string(body))

	// Отправка успешного ответа
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("POST request received successfully"))
}

// backStatistic обрабатывает HTTP-запрос и создает статистический отчет на основе полученных данных.
func backStatistic(w http.ResponseWriter, r *http.Request) {
	// Выводит в консоль сообщение для отладки, указывающее, что функция была вызвана.
	fmt.Println("ya tut")

	// Создание декодера JSON для чтения данных из тела запроса.
	decoder := json.NewDecoder(r.Body)
	var data map[string]interface{}
	err := decoder.Decode(&data)
	if err != nil {
		// В случае ошибки при декодировании JSON, отправляет HTTP-ответ с кодом ошибки и сообщением.
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Проверка наличия ключа "Dimensions" в данных и его приведение к массиву интерфейсов.
	dimensionsArray, ok := data["Dimensions"].([]interface{})
	if !ok {
		// Если ключ "Dimensions" отсутствует или имеет неверный формат, отправляет HTTP-ответ с ошибкой.
		http.Error(w, "Invalid Dimensions format", http.StatusBadRequest)
		return
	}

	// Создание среза строк для хранения измерений.
	dimensions := make([]string, len(dimensionsArray))
	for i, v := range dimensionsArray {
		// Приведение каждого измерения к строковому типу.
		s, ok := v.(string)
		if !ok {
			// Если приведение не удалось, отправляет HTTP-ответ с ошибкой.
			http.Error(w, "Invalid Dimension format", http.StatusBadRequest)
			return
		}
		// Запись измерения в срез dimensions.
		dimensions[i] = s
		fmt.Println("dimensions[i] = s: ", dimensions[i]) // Выводит в консоль текущее измерение для отладки.
	}

	// Создание первого элемента отчета с использованием NewCreatorForJSON.
	creator := NewCreatorForJSON(dimensions[0])

	// Создание дочерних элементов для оставшихся измерений.
	//var currentElement ReportElement = creator
	for _, dim := range dimensions[1:] {
		child := &ChildrenElement{Dimension: dim, Report: []map[string]interface{}{}}
		currentElement := child
		creator.ReportElements = append(creator.ReportElements, currentElement)
	}

	// Получение порядка детализаций из запроса.
	var detailsOrder []string
	if details, ok := data["Details"].([]interface{}); ok {
		for _, detail := range details {
			if d, ok := detail.(string); ok {
				detailsOrder = append(detailsOrder, d)
			}
		}
	}

	// Создание отчета и отправка его в виде JSON-ответа.
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
