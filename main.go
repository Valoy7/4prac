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
type ServerDisconnectedError struct {
	message string
}

func (e *ServerDisconnectedError) Error() string {
	return e.message
}
var count int
///////////////////4 ptakt
type StatData struct {
	ID     int    `json:"Id"`
	PID    *int   `json:"Pid,omitempty"`
	URL    *string `json:"URL,omitempty"`
	SourceIP *string `json:"SourceIP,omitempty"`
	TimeInterval *string `json:"TimeInterval,omitempty"`
	Count  int    `json:"Count"`
}

type ReportElement interface {
	addToReport(Pid int, currStats map[string]string) int
}

type ParentElement struct {
	Id          int
	Pid         interface{}
	URL         interface{}
	SourceIP    interface{}
	TimeInterval interface{}
	Count       int
	deminsion   string
	report      []ReportElement
}

func (pe *ParentElement) addToReport(Pid int, currStats map[string]string) int {
	myStat := currStats[pe.deminsion]
	for _, i := range pe.report {
		if val, ok := i.(*ParentElement); ok && val.deminsion == myStat {
			val.Count++
			return val.Id
		}
	}

	newElement := &ParentElement{
		Id:          len(pe.report),
		Pid:         nil,
		URL:         nil,
		SourceIP:    nil,
		TimeInterval: nil,
		Count:       1,
		deminsion:   myStat,
		report:      nil,
	}
	pe.report = append(pe.report, newElement)
	return newElement.Id
}

type ChildrenElement struct {
	Id          int
	Pid         interface{}
	URL         interface{}
	SourceIP    interface{}
	TimeInterval interface{}
	Count       int
	deminsion   string
	report      []ReportElement
}

func (ce *ChildrenElement) addToReport(Pid int, currStats map[string]string) int {
	myStat := currStats[ce.deminsion]
	for _, i := range ce.report {
		if val, ok := i.(*ChildrenElement); ok && val.deminsion == myStat && val.Pid == Pid {
			val.Count++
			return val.Id
		}
	}

	newElement := &ChildrenElement{
		Id:          len(ce.report),
		Pid:         Pid,
		URL:         nil,
		SourceIP:    nil,
		TimeInterval: nil,
		Count:       1,
		deminsion:   myStat,
		report:      nil, // You may need to initialize it depending on your use case
	}
	ce.report = append(ce.report, newElement)
	return newElement.Id
}

type ReportCreator interface {
	createReport() []StatData
}

type CreatorForJSON struct {
	report    []ReportElement
	deminsion string
}

func (cfj *CreatorForJSON) createReport() []StatData {
	// Инициализация пустого массива для хранения результирующей статистики
	var result []StatData

	// Итерация по элементам в отчете (cfj.report)
	for _, element := range cfj.report {
		// Проверяем, является ли элемент типа *ParentElement
		if val, ok := element.(*ParentElement); ok {
			// Создаем новую структуру StatData, используя данные из *ParentElement
			statData := StatData{
				ID:           val.Id,
				URL:          nil, // Добавьте реальные данные, если они есть
				SourceIP:     nil, // Добавьте реальные данные, если они есть
				TimeInterval: nil, // Добавьте реальные данные, если они есть
				Count:        val.Count,
			}
			// Добавляем созданную структуру в массив result
			result = append(result, statData)
		} else if val, ok := element.(*ChildrenElement); ok {
			// Если элемент типа *ChildrenElement, то используем утверждение (assertion),
			// чтобы получить значение Pid в нужном формате (int)
			pid, ok := val.Pid.(int)
			if !ok {
				// Если утверждение не удалось, выводим ошибку и переходим к следующему элементу
				fmt.Println("Error asserting Pid to int")
				continue
			}
			// Создаем новую структуру StatData, используя данные из *ChildrenElement
			statData := StatData{
				ID:           val.Id,
				PID:          &pid,
				URL:          nil, // Добавьте реальные данные, если они есть
				SourceIP:     nil, // Добавьте реальные данные, если они есть
				TimeInterval: nil, // Добавьте реальные данные, если они есть
				Count:        val.Count,
			}
			// Добавляем созданную структуру в массив result
			result = append(result, statData)
		}
	}

	// Возвращаем сформированный массив структур StatData
	return result
}

func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем данные из тела запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	countStr := strconv.Itoa(count)
	if count == 0 {

		dimensions := strings.Join([]string{countStr, string(body)}, " ")
		// отправляем пост запрос на БД для заполнения таблицы
		resp, err := http.Post("http://localhost:8082/post", "text/plain", strings.NewReader(dimensions))
		if err != nil {
			fmt.Println("Error sending POST request to the second server:", err)
			return
		}
		// увеличиваем каунт. Каунт для удобной ориентации в ХТ.
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
	// Выводим полученные данные
	fmt.Println("Received data:", string(body))

	// Отправляем успешный ответ
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("POST request received successfully"))
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
	fmt.Println("ya tut")
////////////////////////
conn, err := net.Dial("tcp", "localhost:6379")
if err != nil {
	// Ошибка при подключении. Проверяем, является ли ошибка "connection refused".
	// Если да, считаем, что сервер отключен и возвращаем нашу ошибку
	if strings.Contains(err.Error(), "dial tcp [::1]:6379: connectex: No connection could be made because the target machine actively refused it.") {
		err = &ServerDisconnectedError{"Сервер отключен, попробуйте позже!"}
		fmt.Println("Сервер отключен, попробуйте позже!")
		return
	}
	// Другая ошибка - выводим ее и завершаем работу
	fmt.Println("Error connecting:", err)
	return
}
defer conn.Close() // Всегда закрываем соединение в конце работы функции
reader := bufio.NewReader(conn)
writer := bufio.NewWriter(conn)

_, err = writer.WriteString("HGETS count" + "\n")
if err != nil {
	fmt.Println("Error writing to server:", err)
	return
}
writer.Flush()
counterStr, err := reader.ReadString('\n')
if err != nil {
    fmt.Println("Error reading counter:", err)
    return
}

counterStr = strings.TrimSuffix(counterStr, "\n")

counter, err := strconv.Atoi(counterStr)
if err != nil {
    fmt.Println("Error converting counter to int:", err)
    return
}

for i := 0; i < counter; i++ {
    istr := strconv.Itoa(i)
    _, err = writer.WriteString("HGETS " + istr + "\n")
    if err != nil {
        fmt.Println("Error writing to server:", err)
        return
    }
    writer.Flush()

dimensions, _ := reader.ReadString('\n')     // Чтение и отправка команды от пользователя
dimensions = strings.TrimSpace(dimensions) // Удаляем пробелы и символы новой строки
	
	fmt.Println("dimensions:", dimensions)
	// Преобразование JSON-строки в массив измерений
var dimensionsData map[string][]string
if err := json.Unmarshal([]byte(dimensions), &dimensionsData); err != nil {
    fmt.Println("Error decoding dimensions JSON:", err)
    return
}

// Получение измерений
dimensionsList, ok := dimensionsData["Dimensions"]
if !ok {
    fmt.Println("Missing 'Dimensions' key in dimensions JSON")
    return
}

// Создание экземпляра CreatorForJSON
creator := &CreatorForJSON{report: nil, deminsion: strings.Join(dimensionsList, "")}

// Перебор измерений и добавление их в статистику
for _, dem := range dimensionsList {
    // Пример: создание экземпляра ParentElement и добавление в статистику
    parentElement := &ParentElement{
        Id:          0, // Уточните, какие значения должны быть установлены по умолчанию
        Pid:         nil,
        URL:         nil,
        SourceIP:    nil,
        TimeInterval: nil,
        Count:       0,
        deminsion:   dem,
        report:      nil, // Уточните, как инициализировать report
    }

    // Пример: добавление элемента в статистику через CreatorForJSON
    creator.report = append(creator.report, parentElement)

    // То же самое может быть сделано для ChildrenElement, в зависимости от вашей логики
}

// Создание отчета
stats := creator.createReport()

// Теперь у вас есть статистика, сформированная на основе измерений
fmt.Println("Stats based on dimensions:", stats)
//здесь мы должны из данных dimensions формировать статистику, используя функции и классы выше
}
///////////////////
    // Читаем данные из тела запроса
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Error reading request body", http.StatusInternalServerError)
        return
    }

    // Декодируем JSON-данные
    var stats []StatData
    err = json.Unmarshal(body, &stats)
    if err != nil {
        http.Error(w, "Error decoding JSON data", http.StatusInternalServerError)
        return
    }

    // Делаем что-то с полученной статистикой, например, выводим её
    fmt.Println("Received stats:", stats)
 // после формирования статистики, отправляем ее обратно тому, кто ее запрашивал
    // Отправляем успешный ответ
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Stats received successfully"))
}

func main() {
    count = 0
    // Настраиваем обработчик для POST-запросов по пути "/post"
    http.HandleFunc("/post", handlePostRequest)
    http.HandleFunc("/report", handleReport) // Добавляем обработчик для /report
    // Запускаем веб-сервер на порту 8081
    err := http.ListenAndServe(":8081", nil)
    if err != nil {
        fmt.Println("Error starting server:", err)
    }
}