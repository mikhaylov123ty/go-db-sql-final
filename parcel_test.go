package main

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// randSource источник псевдо случайных чисел.
	// Для повышения уникальности в качестве seed
	// используется текущее время в unix формате (в виде числа)
	randSource = rand.NewSource(time.Now().UnixNano())
	// randRange использует randSource для генерации случайных чисел
	randRange = rand.New(randSource)
)

// getTestConnection возвращает соединение для тестовых запросов
// ParcelStore - указатель на структуру sql.DB
// *Parcel - указатель на структуру Parcel
func getTestConnection() (ParcelStore, *Parcel, error) {
	// инициализируем подключение к ДБ
	conn, err := sql.Open("sqlite", "./tracker.db")
	if err != nil {
		return ParcelStore{}, &Parcel{}, err
	}

	// возвращаем указатели на структуры
	return NewParcelStore(conn), getTestParcel(), nil
}

// getTestParcel возвращает указатель на структуру тестовой посылки
func getTestParcel() *Parcel {
	return &Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// TestAddGetDelete проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	//prepare
	store, parcel, err := getTestConnection()
	require.NoError(t, err)
	defer store.db.Close()

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	id, err := store.Add(*parcel)
	require.NoError(t, err)
	assert.NotZero(t, id, "Error ID is not Zero")

	// get
	// получите только что добавленную посылку, убедитесь в отсутствии ошибки
	p, err := store.Get(id)
	require.NoError(t, err)

	// проверьте, что значения всех полей в полученном объекте совпадают со значениями полей в переменной parcel
	parcel.Number = id
	assert.Equal(t, p, *parcel)

	// delete
	// удалите добавленную посылку, убедитесь в отсутствии ошибки
	err = store.Delete(id)
	require.NoError(t, err)

	// проверьте, что посылку больше нельзя получить из БД
	_, err = store.Get(id)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestSetAddress проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	// prepare
	store, parcel, err := getTestConnection()
	require.NoError(t, err)
	defer store.db.Close()

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	id, err := store.Add(*parcel)
	require.NoError(t, err)
	assert.NotZero(t, id)

	// set address
	// обновите адрес, убедитесь в отсутствии ошибки
	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	require.NoError(t, err)

	// check
	// получите добавленную посылку и убедитесь, что адрес обновился
	p, err := store.Get(id)
	require.NoError(t, err)
	assert.Equal(t, newAddress, p.Address)

	// доп. проверка, что адрес не меняется при смене статуса на любой кроме "registered"
	// set status
	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	// set address
	// снова обновляем адрес, необходимо убедиться, что адрес не меняется, т.к. статус "sent"
	newAddress = "new test address on sent parcel"
	err = store.SetAddress(id, newAddress)
	require.ErrorIs(t, err, sql.ErrNoRows, "should be no rows affected")
}

// TestSetStatus проверяет обновление статуса
func TestSetStatus(t *testing.T) {
	// prepare
	store, parcel, err := getTestConnection()
	require.NoError(t, err)
	defer store.db.Close()

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	id, err := store.Add(*parcel)
	require.NoError(t, err)
	assert.NotZero(t, id)

	// set status
	// обновите статус, убедитесь в отсутствии ошибки
	err = store.SetStatus(id, ParcelStatusDelivered)
	require.NoError(t, err)

	// check
	// получите добавленную посылку и убедитесь, что статус обновился
	p, err := store.Get(id)
	require.NoError(t, err)
	assert.Equal(t, ParcelStatusDelivered, p.Status)
}

// TestGetByClient проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	// prepare
	store, _, err := getTestConnection()
	require.NoError(t, err)
	defer store.db.Close()

	parcels := []Parcel{
		*getTestParcel(),
		*getTestParcel(),
		*getTestParcel(),
	}
	parcelMap := map[int]Parcel{}

	// задаём всем посылкам один и тот же идентификатор клиента
	client := randRange.Intn(10_000_000)
	parcels[0].Client = client
	parcels[1].Client = client
	parcels[2].Client = client

	// add
	for i := 0; i < len(parcels); i++ {
		// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
		id, err := store.Add(parcels[i])
		require.NoError(t, err)

		// обновляем идентификатор добавленной у посылки
		parcels[i].Number = id

		// сохраняем добавленную посылку в структуру map, чтобы её можно было легко достать по идентификатору посылки
		parcelMap[id] = parcels[i]
	}

	// get by client
	// получите список посылок по идентификатору клиента, сохранённого в переменной client
	storedParcels, err := store.GetByClient(client)

	// убедитесь в отсутствии ошибки
	require.NoError(t, err)

	// убедитесь, что количество полученных посылок совпадает с количеством добавленных
	assert.Len(t, storedParcels, len(parcels))

	// check
	for _, parcel := range storedParcels {
		// в parcelMap лежат добавленные посылки, ключ - идентификатор посылки, значение - сама посылка
		// убедитесь, что все посылки из storedParcels есть в parcelMap
		assert.Contains(t, parcelMap, parcel.Number)

		// убедитесь, что значения полей полученных посылок заполнены верно
		assert.Equal(t, parcelMap[parcel.Number], parcel)
	}
}
