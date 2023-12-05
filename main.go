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

// /////////////////4 ptakt
type StatData struct {
	ID           int     `json:"Id"`
	PID          *int    `json:"Pid,omitempty"`
	URL          *string `json:"URL,omitempty"`
	SourceIP     *string `json:"SourceIP,omitempty"`
	TimeInterval *string `json:"TimeInterval,omitempty"`
	Count        int     `json:"Count"`
}

type ReportElement interface {
	addToReport(Pid int, currStats map[string]string) int
}

type ParentElement struct {
	Id           int
	Pid          interface{}
	URL          interface{}
	SourceIP     interface{}
	TimeInterval interface{}
	Count        int
	deminsion    string
	report       []ReportElement
}

// addToReport добавляет статистический элемент в отчет (рекурсивно обрабатывает элементы дерева отчета).
// Возвращает идентификатор нового или существующего элемента, который был добавлен в отчет.
func (pe *ParentElement) addToReport(Pid int, currStats map[string]string) int {
	// Извлекаем значение измерения текущего элемента
	myStat := currStats[pe.deminsion]

	// Итерируем по элементам отчета текущего уровня (родителя)
	for _, i := range pe.report {
		// Проверяем, является ли элемент типа *ParentElement и соответствует ли измерение
		if val, ok := i.(*ParentElement); ok && val.deminsion == myStat {
			// Увеличиваем счетчик и возвращаем идентификатор существующего элемента
			val.Count++
			return val.Id
		}
	}

	// Создаем новый элемент отчета, так как элемент с заданным измерением не найден
	newElement := &ParentElement{
		Id:           len(pe.report),
		Pid:          nil,    // Идентификатор родительского элемента (может потребоваться уточнение)
		URL:          nil,    // Данные URL (добавьте реальные данные, если они есть)
		SourceIP:     nil,    // Данные SourceIP (добавьте реальные данные, если они есть)
		TimeInterval: nil,    // Данные TimeInterval (добавьте реальные данные, если они есть)
		Count:        1,      // Устанавливаем начальное значение счетчика в 1
		deminsion:    myStat, // Измерение текущего элемента
		report:       nil,    // Инициализируйте его элементами отчета (возможно, потребуется уточнение)
	}

	// Добавляем новый элемент к отчету текущего родительского элемента
	pe.report = append(pe.report, newElement)

	// Возвращаем идентификатор нового элемента
	return newElement.Id
}

// ChildrenElement - это элемент отчета, представляющий статистику с дополнительными измерениями.
type ChildrenElement struct {
	Id           int             // Идентификатор элемента
	Pid          interface{}     // Идентификатор родительского элемента
	URL          interface{}     // Данные URL (добавьте реальные данные, если они есть)
	SourceIP     interface{}     // Данные SourceIP (добавьте реальные данные, если они есть)
	TimeInterval interface{}     // Данные TimeInterval (добавьте реальные данные, если они есть)
	Count        int             // Счетчик элемента
	deminsion    string          // Измерение текущего элемента
	report       []ReportElement // Элементы отчета текущего элемента
}

// // addToReport - это метод типа дочернего элемента, который добавляет новый элемент отчета или увеличивает количество
// существующего элемента на основе предоставленного Pid (родительского идентификатора) и текущей статистики.
// Он возвращает идентификатор добавленного или обновленного элемента отчета.
func (ce *ChildrenElement) addToReport(Pid int, currStats map[string]string) int {
	// Извлеките значение измерения для текущего ChildrenElement из предоставленной статистики
	myStat := currStats[ce.deminsion]

	// Выполните итерацию по существующим элементам отчета, чтобы найти соответствие на основе измерения, Pid и утверждения типа
	for _, i := range ce.report {
		if val, ok := i.(*ChildrenElement); ok && val.deminsion == myStat && val.Pid == Pid {
			// If a match is found, increment the count and return the ID of the matched element
			val.Count++
			return val.Id
		}
	}

	// Если совпадение не найдено, создайте новый дочерний элемент и добавьте его в отчет
	newElement := &ChildrenElement{
		Id:           len(ce.report),
		Pid:          Pid,
		URL:          nil,
		SourceIP:     nil,
		TimeInterval: nil,
		Count:        1,
		deminsion:    myStat,
		report:       nil, // You may need to initialize it depending on your use case
	}
	ce.report = append(ce.report, newElement)

	// Возвращает идентификатор вновь добавленного элемента
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
		fmt.Println("ya zdesya")
		fmt.Println(" eto element: ", element)
		// Проверяем, является ли элемент типа *ParentElement
		if val, ok := element.(*ParentElement); ok {
			// Создаем новую структуру StatData, используя данные из *ParentElement
			statData := StatData{
				ID:           val.Id,
				URL:          nil, //
				SourceIP:     nil, //
				TimeInterval: nil, //
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
				URL:          nil, //
				SourceIP:     nil, //
				TimeInterval: nil, //
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

	_, err = writer.WriteString("HGETSC" + "\n")
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

	if counterStr == "" {
		fmt.Println("Empty counter string received")
		return
	}

	counter, err := strconv.Atoi(counterStr)
	if err != nil {
		fmt.Println("Error converting counter to int:", err)
		return
	}
	fmt.Println("Counter ", counter)
	for i := 0; i <= counter; i++ {

		istr := strconv.Itoa(i)
		fmt.Println("Counter!! ", istr)
		_, err = writer.WriteString("HGETS " + istr + "\n")
		if err != nil {
			fmt.Println("Error writing to server:", err)
			return
		}
		writer.Flush()

		dimensions, _ := reader.ReadString('\n')   // Чтение и отправка команды от пользователя
		dimensions = strings.TrimSpace(dimensions) // Удаляем пробелы и символы новой строки

		fmt.Println("dimensions:", dimensions)

		// Чтение измерений из строки и разделение их по пробелу
		dimensionsList := strings.Split(dimensions, " ")
		// Создание экземпляра CreatorForJSON
		creator := &CreatorForJSON{report: nil, deminsion: strings.Join(dimensionsList, " ")}

		// Перебор измерений и добавление их в статистику
		for _, dem := range dimensionsList {
			// Пример: создание экземпляра ParentElement и добавление в статистику
			parentElement := &ParentElement{
				Id:           0, //
				Pid:          nil,
				URL:          nil,
				SourceIP:     nil,
				TimeInterval: nil,
				Count:        0,
				deminsion:    dem,
				report:       nil,
			}

			// добавление элемента в статистику через CreatorForJSON
			creator.report = append(creator.report, parentElement)

		}

		// Создание отчета
		stats := creator.createReport()

		fmt.Println("Stats based on dimensions:", stats)

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
