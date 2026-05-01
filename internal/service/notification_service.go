package service

type NotificationService interface {
}

type notificationService struct {
}

func NewNotificationService() NotificationService {
	return &notificationService{}

}

func (s *notificationService) SendNewReservationAlert(reservationID string) error {
	return nil
}

func (s *notificationService) SendBookingConfirmation(reservationID string) error {

	return nil
}

func (s *notificationService) SendNewQueueAlert(restaurantID string) error {
	return nil
}
