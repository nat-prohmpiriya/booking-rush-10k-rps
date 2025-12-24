export default () => ({
  port: parseInt(process.env.PORT || '8085', 10),
  environment: process.env.NODE_ENV || 'development',

  // MongoDB
  mongodb: {
    uri:
      process.env.MONGODB_URI ||
      'mongodb://localhost:27017/booking_rush_notifications',
  },

  // Kafka
  kafka: {
    brokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
    clientId: process.env.KAFKA_CLIENT_ID || 'notification-service',
    groupId: process.env.KAFKA_GROUP_ID || 'notification-service-group',
  },

  // Resend (Email)
  resend: {
    apiKey: process.env.RESEND_API_KEY || '',
    fromEmail:
      process.env.FROM_EMAIL || 'Booking Rush <onboarding@resend.dev>',
  },

  // Service URLs (for fetching additional data if needed)
  services: {
    authServiceUrl:
      process.env.AUTH_SERVICE_URL || 'http://localhost:8081/api/v1',
    ticketServiceUrl:
      process.env.TICKET_SERVICE_URL || 'http://localhost:8082/api/v1',
  },
});
