import axios, { AxiosInstance } from 'axios';
import { Telemetry, FleetMetrics, Vehicle, Trip, Maintenance, Cost, TimeRange } from '../types';

const API_BASE_URL =
  process.env.REACT_APP_API_URL ||
  (typeof window !== 'undefined' && window.location && window.location.hostname !== 'localhost'
    ? window.location.origin
    : 'http://localhost:8080');

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
          // Only handle unauthorized access for this specific application
          // Check if the error is from our API base URL to avoid affecting other apps
          if (error.config?.url?.includes(API_BASE_URL) || error.config?.baseURL?.includes(API_BASE_URL)) {
            localStorage.removeItem('auth_token');
            localStorage.removeItem('user');
            // Only redirect if we're on the fleet sustainability domain
            if (window.location.hostname === 'localhost' && window.location.port === '3000') {
              window.location.href = '/login';
            }
          }
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
    params.append('sort', 'asc');
    params.append('limit', '0');

    const response = await this.api.get(`/api/telemetry?${params.toString()}`);
    return response.data || [];
  }

  async getTelemetryByVehicle(vehicleId: string, timeRange?: TimeRange): Promise<Telemetry[]> {
    const params = new URLSearchParams();
    if (timeRange?.from) params.append('from', timeRange.from);
    if (timeRange?.to) params.append('to', timeRange.to);
    params.append('vehicle_id', vehicleId);
    params.append('sort', 'asc');
    params.append('limit', '0');
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

  async getAdvancedMetrics(timeRange?: TimeRange): Promise<any> {
    const params = new URLSearchParams();
    if (timeRange?.from) params.append('from', timeRange.from);
    const response = await this.api.get(`/api/telemetry/metrics/advanced?${params.toString()}`);
    return response.data || {};
  }

  // Vehicles endpoints
  async getVehicles(timeRange?: TimeRange): Promise<Vehicle[]> {
    const params = new URLSearchParams();
    if (timeRange?.from) params.append('from', timeRange.from);
    if (timeRange?.to) params.append('to', timeRange.to);
    
    const response = await this.api.get(`/api/vehicles?${params.toString()}`);
    return response.data || [];
  }

  // Alerts
  async getAlerts(timeRange?: TimeRange): Promise<Array<{type:string; vehicle_id:string; value:number; ts:string}>> {
    const params = new URLSearchParams();
    if (timeRange?.from) params.append('from', timeRange.from);
    if (timeRange?.to) params.append('to', timeRange.to);
    const response = await this.api.get(`/api/alerts?${params.toString()}`);
    return response.data || [];
  }

  async createVehicle(vehicle: Omit<Vehicle, 'id'>): Promise<{ id: string; message: string }> {
    const response = await this.api.post('/api/vehicles', vehicle);
    return response.data;
  }

  async updateVehicle(id: string, vehicle: Partial<Vehicle>): Promise<{ id: string; message: string }> {
    const response = await this.api.put(`/api/vehicles/${id}`, vehicle);
    return response.data;
  }

  async deleteVehicle(id: string): Promise<{ id: string; message: string }> {
    const response = await this.api.delete(`/api/vehicles/${id}`);
    return response.data;
  }

  // Trip methods
  async getTrips(timeRange?: TimeRange): Promise<Trip[]> {
    const params = timeRange ? { from: timeRange.from, to: timeRange.to } : {};
    const response = await this.api.get('/api/trips', { params });
    return response.data || [];
  }

  async postTrip(trip: Omit<Trip, 'id'>): Promise<{ id: string; message: string }> {
    const response = await this.api.post('/api/trips', trip);
    return response.data;
  }

  async updateTrip(id: string, trip: Partial<Trip>): Promise<{ id: string; message: string }> {
    const response = await this.api.put(`/api/trips/${id}`, trip);
    return response.data;
  }

  async deleteTrip(id: string): Promise<{ id: string; message: string }> {
    const response = await this.api.delete(`/api/trips/${id}`);
    return response.data;
  }

  // Maintenance methods
  async getMaintenance(timeRange?: TimeRange): Promise<Maintenance[]> {
    const params = timeRange ? { from: timeRange.from, to: timeRange.to } : {};
    const response = await this.api.get('/api/maintenance', { params });
    return response.data || [];
  }

  async postMaintenance(maintenance: Omit<Maintenance, 'id'>): Promise<{ id: string; message: string }> {
    const response = await this.api.post('/api/maintenance', maintenance);
    return response.data;
  }

  async updateMaintenance(id: string, maintenance: Partial<Maintenance>): Promise<{ id: string; message: string }> {
    const response = await this.api.put(`/api/maintenance/${id}`, maintenance);
    return response.data;
  }

  async deleteMaintenance(id: string): Promise<{ id: string; message: string }> {
    const response = await this.api.delete(`/api/maintenance/${id}`);
    return response.data;
  }

  // Cost methods
  async getCosts(timeRange?: TimeRange): Promise<Cost[]> {
    const params = timeRange ? { from: timeRange.from, to: timeRange.to } : {};
    const response = await this.api.get('/api/costs', { params });
    return response.data || [];
  }

  async postCost(cost: Omit<Cost, 'id'>): Promise<{ id: string; message: string }> {
    const response = await this.api.post('/api/costs', cost);
    return response.data;
  }

  async updateCost(id: string, cost: Partial<Cost>): Promise<{ id: string; message: string }> {
    const response = await this.api.put(`/api/costs/${id}`, cost);
    return response.data;
  }

  async deleteCost(id: string): Promise<{ id: string; message: string }> {
    const response = await this.api.delete(`/api/costs/${id}`);
    return response.data;
  }

  // Authentication
  async login(username: string, password: string): Promise<{ token: string; user: any }> {
    const response = await this.api.post('/api/auth/login', { username, password });
    return response.data;
  }

  async register(userData: any): Promise<{ token: string; user: any }> {
    const response = await this.api.post('/api/auth/register', userData);
    return response.data;
  }

  async getProfile(): Promise<any> {
    const response = await this.api.get('/api/auth/profile');
    return response.data;
  }

  async updateProfile(profileData: any): Promise<any> {
    const response = await this.api.put('/api/auth/profile', profileData);
    return response.data;
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<any> {
    const response = await this.api.post('/api/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    });
    return response.data;
  }

  logout(): void {
    // Only clear fleet sustainability specific items
    localStorage.removeItem('auth_token');
    localStorage.removeItem('user');
    // Don't clear other localStorage items that might be used by other applications
  }

  isAuthenticated(): boolean {
    return !!localStorage.getItem('auth_token');
  }

  getCurrentUser(): any {
    const userStr = localStorage.getItem('user');
    return userStr ? JSON.parse(userStr) : null;
  }
}

export const apiService = new ApiService();
export default apiService; 