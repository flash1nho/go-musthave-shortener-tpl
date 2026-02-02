package solution

import (
	"testing"
)

// MockObject — тестовая структура для проверки сброса состояния.
type MockObject struct {
	Value int
}

func (m *MockObject) Reset() {
	m.Value = 0
}

func TestPool(t *testing.T) {
	// Создаем пул с фабрикой новых объектов
	factory := func() *MockObject {
		return &MockObject{Value: 42}
	}
	pool := New[*MockObject](factory)

	// 1. Проверяем получение нового объекта
	obj1 := pool.Get()
	if obj1.Value != 42 {
		t.Errorf("expected 42, got %d", obj1.Value)
	}

	// 2. Изменяем состояние и возвращаем в пул
	obj1.Value = 100
	pool.Put(obj1)

	// 3. Получаем объект снова.
	// Он должен быть тем же (или новым), но обязательно со сброшенным значением.
	obj2 := pool.Get()
	if obj2.Value != 0 {
		t.Errorf("expected Reset to set Value to 0, got %d", obj2.Value)
	}
}

func TestPoolConcurrent(t *testing.T) {
	pool := New[*MockObject](func() *MockObject { return &MockObject{} })

	// Простая проверка на отсутствие race condition при параллельной работе
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				obj := pool.Get()
				obj.Value = j
				pool.Put(obj)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
