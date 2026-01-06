import {
  Injectable,
  Logger,
  OnModuleInit,
  OnModuleDestroy,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Kafka, Consumer, EachMessagePayload, CompressionTypes, CompressionCodecs } from 'kafkajs';
import SnappyCodec from 'kafkajs-snappy';

// Register Snappy codec for KafkaJS
CompressionCodecs[CompressionTypes.Snappy] = SnappyCodec;
import { BookingEventHandler } from './handlers/booking-event.handler';
import {
  EventType,
  PaymentSuccessEvent,
  BookingExpiredEvent,
  BookingCancelledEvent,
} from './dto/events.dto';

@Injectable()
export class KafkaConsumerService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(KafkaConsumerService.name);
  private kafka: Kafka;
  private consumer: Consumer;
  private isConnected = false;

  // Topics to subscribe
  private readonly topics = [
    'payment.success',
    'booking.expired',
    'booking.cancelled',
  ];

  constructor(
    private readonly configService: ConfigService,
    private readonly bookingEventHandler: BookingEventHandler,
  ) {
    const brokers = this.configService.get<string[]>('kafka.brokers') || [
      'localhost:9092',
    ];
    const clientId =
      this.configService.get<string>('kafka.clientId') || 'notification-service';
    const groupId =
      this.configService.get<string>('kafka.groupId') ||
      'notification-service-group';

    this.kafka = new Kafka({
      clientId,
      brokers,
      retry: {
        initialRetryTime: 100,
        retries: 8,
      },
    });

    this.consumer = this.kafka.consumer({ groupId });
  }

  async onModuleInit(): Promise<void> {
    await this.connect();
  }

  async onModuleDestroy(): Promise<void> {
    await this.disconnect();
  }

  async connect(): Promise<void> {
    try {
      await this.consumer.connect();
      this.logger.log('Kafka consumer connected');

      // Subscribe to topics
      for (const topic of this.topics) {
        await this.consumer.subscribe({ topic, fromBeginning: false });
        this.logger.log(`Subscribed to topic: ${topic}`);
      }

      // Start consuming
      await this.consumer.run({
        eachMessage: async (payload) => {
          await this.handleMessage(payload);
        },
      });

      this.isConnected = true;
      this.logger.log('Kafka consumer started');
    } catch (error) {
      this.logger.error(`Failed to connect Kafka consumer: ${error.message}`);
      // Don't throw - allow service to start without Kafka
    }
  }

  async disconnect(): Promise<void> {
    if (this.isConnected) {
      try {
        await this.consumer.disconnect();
        this.isConnected = false;
        this.logger.log('Kafka consumer disconnected');
      } catch (error) {
        this.logger.error(`Error disconnecting Kafka consumer: ${error.message}`);
      }
    }
  }

  private async handleMessage(payload: EachMessagePayload): Promise<void> {
    const { topic, partition, message } = payload;
    const key = message.key?.toString();
    const value = message.value?.toString();

    this.logger.debug(
      `Received message on ${topic}[${partition}]: key=${key}`,
    );

    if (!value) {
      this.logger.warn(`Empty message received on ${topic}`);
      return;
    }

    try {
      const event = JSON.parse(value);

      switch (topic) {
        case 'payment.success':
          await this.bookingEventHandler.handlePaymentSuccess(
            event as PaymentSuccessEvent,
          );
          break;

        case 'booking.expired':
          await this.bookingEventHandler.handleBookingExpired(
            event as BookingExpiredEvent,
          );
          break;

        case 'booking.cancelled':
          await this.bookingEventHandler.handleBookingCancelled(
            event as BookingCancelledEvent,
          );
          break;

        default:
          this.logger.warn(`Unknown topic: ${topic}`);
      }
    } catch (error) {
      this.logger.error(
        `Error processing message on ${topic}: ${error.message}`,
        error.stack,
      );
      // Don't throw - let Kafka continue with next message
    }
  }

  /**
   * Check if consumer is connected
   */
  isHealthy(): boolean {
    return this.isConnected;
  }
}
