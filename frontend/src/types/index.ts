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