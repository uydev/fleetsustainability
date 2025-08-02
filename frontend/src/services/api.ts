import axios, { AxiosInstance } from 'axios';
import { Telemetry, FleetMetrics, Vehicle, TimeRange } from '../types';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

class ApiService {
  private api: AxiosInstance;

  constructor() {
    this.api = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request interceptor for authentication
    this.api.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('auth_token');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => {
        return Promise.reject(error);
      }
    );

    // Add response interceptor for error handling
    this.api.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          // Handle unauthorized access
          localStorage.removeItem('auth_token');
          window.location.href = '/login';
        }
        return Promise.reject(error);
      }
    );
  }

  // Telemetry endpoints
  async getTelemetry(timeRange?: TimeRange): Promise<Telemetry[]> {
    const params = new URLSearchParams();
    if (timeRange?.from) params.append('from', timeRange.from);
    if (timeRange?.to) params.append('to', timeRange.to);

    const response = await this.api.get(`/api/telemetry?${params.toString()}`);
    return response.data || [];
  }

  async postTelemetry(telemetry: Omit<Telemetry, 'id'>): Promise<void> {
    await this.api.post('/api/telemetry', telemetry);
  }

  // Metrics endpoints
  async getFleetMetrics(timeRange?: TimeRange): Promise<FleetMetrics> {
    const params = new URLSearchParams();
    if (timeRange?.from) params.append('from', timeRange.from);
    if (timeRange?.to) params.append('to', timeRange.to);

    const response = await this.api.get(`/api/telemetry/metrics?${params.toString()}`);
    return response.data || { total_emissions: 0, ev_percent: 0, total_records: 0 };
  }

  // Vehicles endpoints
  async getVehicles(): Promise<Vehicle[]> {
    const response = await this.api.get('/api/vehicles');
    return response.data;
  }

  // Authentication
  async login(token: string): Promise<void> {
    localStorage.setItem('auth_token', token);
  }

  logout(): void {
    localStorage.removeItem('auth_token');
  }

  isAuthenticated(): boolean {
    return !!localStorage.getItem('auth_token');
  }
}

export const apiService = new ApiService();
export default apiService; 