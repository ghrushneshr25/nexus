package main

func NewUserRepository(config Config) UserRepository {
	return userRepository{
		config: config,
	}
}

func NewUserService(repository UserRepository) UserService {
	return userService{
		repository: repository,
	}
}

func NewOrdersClient() MessageClient {
	return messageClient{
		name: "orders",
	}
}

func NewAuditClient() MessageClient {
	return messageClient{
		name: "audit",
	}
}

func NewAuditHandler() EventHandler {
	return auditHandler{}
}

func NewMetricsHandler() EventHandler {
	return metricsHandler{}
}
