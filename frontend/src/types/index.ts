export interface Location {
  lat: number;
  lon: number;
}

export interface Telemetry {
  id?: string;
  vehicle_id: string;
  timestamp: string;
  location: Location;
  speed: number;
  fuel_level?: number;
  battery_level?: number;
  emissions: number;
  type: 'ICE' | 'EV';
  status: 'active' | 'inactive';
}

export interface Trip {
  id?: string;
  vehicle_id: string;
  driver_id: string;
  start_location: Location;
  end_location: Location;
  start_time: string;
  end_time: string;
  distance: number;
  duration: number;
  fuel_consumption: number;
  battery_consumption: number;
  cost: number;
  purpose: 'business' | 'personal' | 'delivery';
  status: 'planned' | 'in_progress' | 'completed' | 'cancelled';
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface Maintenance {
  id?: string;
  vehicle_id: string;
  service_type: 'oil_change' | 'tire_rotation' | 'brake_service' | 'battery_service' | 'inspection';
  description: string;
  service_date: string;
  next_service_date: string;
  mileage: number;
  cost: number;
  labor_cost: number;
  parts_cost: number;
  technician: string;
  service_location: string;
  status: 'scheduled' | 'in_progress' | 'completed' | 'cancelled';
  priority: 'low' | 'medium' | 'high' | 'critical';
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface Cost {
  id?: string;
  vehicle_id: string;
  category: 'fuel' | 'maintenance' | 'insurance' | 'registration' | 'tolls' | 'parking' | 'other';
  description: string;
  amount: number;
  date: string;
  invoice_number: string;
  vendor: string;
  location: string;
  payment_method: 'credit_card' | 'cash' | 'check' | 'electronic';
  status: 'pending' | 'paid' | 'disputed' | 'cancelled';
  receipt_url: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface FleetMetrics {
  total_emissions: number;
  ev_percent: number;
  total_records: number;
}

export interface Vehicle {
  id: string;
  type: 'ICE' | 'EV';
  make?: string;
  model?: string;
  year?: number;
  current_location?: Location;
  status: 'active' | 'inactive';
}

export interface ApiResponse<T> {
  data: T;
  error?: string;
}

export interface TimeRange {
  from?: string;
  to?: string;
}

// Driver efficiency leaderboard types
export interface DriverEfficiency {
  driver_id: string;
  driver_name: string;
  total_distance: number;
  total_fuel_consumed: number;
  total_emissions: number;
  average_speed: number;
  efficiency_score: number;
  fuel_efficiency: number;
  emission_efficiency: number;
  safety_score: number;
  trips_completed: number;
  total_driving_time: number;
  rank: number;
  improvement_since_last_period: number;
  badges: string[];
}

export interface DriverLeaderboard {
  drivers: DriverEfficiency[];
  period: 'daily' | 'weekly' | 'monthly' | 'yearly';
  total_drivers: number;
  last_updated: string;
}