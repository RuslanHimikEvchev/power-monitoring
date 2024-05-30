package core

type RedisStorage struct {
	redis *RedisConnect
}

func NewRedisStorage(redis *RedisConnect) *RedisStorage {
	return &RedisStorage{redis}
}

func (s *RedisStorage) Store(key string, value string) error {
	status := s.redis.GetConnection().Set(key, value, 0)

	if status.Err() != nil {
		return status.Err()
	}

	return nil
}

func (s *RedisStorage) Get(key string) (string, error) {
	cmd := s.redis.GetConnection().Get(key)

	if cmd.Err() != nil {
		return "", cmd.Err()
	}

	r, err := cmd.Result()

	if err != nil {
		return "", err
	}

	return r, nil
}
