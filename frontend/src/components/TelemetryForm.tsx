import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Box,
  Typography,
  Alert,
} from '@mui/material';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { Telemetry } from '../types';

interface TelemetryFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (telemetry: Omit<Telemetry, 'id'>) => Promise<void>;
  title: string;
}

const TelemetryForm: React.FC<TelemetryFormProps> = ({
  open,
  onClose,
  onSubmit,
  title,
}) => {
  const [formData, setFormData] = useState<Omit<Telemetry, 'id'>>({
    vehicle_id: '',
    timestamp: new Date().toISOString(),
    location: { lat: 40.7128, lon: -74.0060 },
    speed: 0,
    fuel_level: undefined,
    battery_level: undefined,
    emissions: 0,
    type: 'ICE',
    status: 'active',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [vehicleType, setVehicleType] = useState<'ICE' | 'EV'>('ICE');

  useEffect(() => {
    if (open) {
      setFormData({
        vehicle_id: '',
        timestamp: new Date().toISOString(),
        location: { lat: 40.7128, lon: -74.0060 },
        speed: 0,
        fuel_level: undefined,
        battery_level: undefined,
        emissions: 0,
        type: 'ICE',
        status: 'active',
      });
      setVehicleType('ICE');
      setErrors({});
    }
  }, [open]);

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.vehicle_id.trim()) {
      newErrors.vehicle_id = 'Vehicle ID is required';
    }
    if (formData.speed < 0 || formData.speed > 300) {
      newErrors.speed = 'Speed must be between 0 and 300 km/h';
    }
    if (formData.emissions < 0) {
      newErrors.emissions = 'Emissions must be non-negative';
    }
    if (vehicleType === 'ICE' && (formData.fuel_level === undefined || formData.fuel_level < 0 || formData.fuel_level > 100)) {
      newErrors.fuel_level = 'Fuel level must be between 0 and 100%';
    }
    if (vehicleType === 'EV' && (formData.battery_level === undefined || formData.battery_level < 0 || formData.battery_level > 100)) {
      newErrors.battery_level = 'Battery level must be between 0 and 100%';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }

    setLoading(true);
    try {
      await onSubmit(formData);
      onClose();
    } catch (error) {
      console.error('Error submitting telemetry:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field: keyof Omit<Telemetry, 'id'>, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const handleVehicleTypeChange = (type: 'ICE' | 'EV') => {
    setVehicleType(type);
    setFormData(prev => ({ 
      ...prev, 
      type,
      battery_level: type === 'EV' ? 80 : undefined,
      fuel_level: type === 'ICE' ? 50 : undefined,
    }));
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <Box display="flex" flexDirection="column" gap={2}>
            <FormControl fullWidth>
              <InputLabel>Vehicle Type</InputLabel>
              <Select
                value={vehicleType}
                onChange={(e) => handleVehicleTypeChange(e.target.value as 'ICE' | 'EV')}
                label="Vehicle Type"
              >
                <MenuItem value="ICE">ICE (Internal Combustion Engine)</MenuItem>
                <MenuItem value="EV">EV (Electric Vehicle)</MenuItem>
              </Select>
            </FormControl>

            <FormControl fullWidth>
              <InputLabel>Status</InputLabel>
              <Select
                value={formData.status}
                onChange={(e) => handleChange('status', e.target.value as 'active' | 'inactive')}
                label="Status"
              >
                <MenuItem value="active">Active</MenuItem>
                <MenuItem value="inactive">Inactive</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Vehicle ID"
              value={formData.vehicle_id}
              onChange={(e) => handleChange('vehicle_id', e.target.value)}
              error={!!errors.vehicle_id}
              helperText={errors.vehicle_id}
              fullWidth
              required
            />

            <LocalizationProvider dateAdapter={AdapterDateFns}>
              <DateTimePicker
                label="Timestamp"
                value={new Date(formData.timestamp)}
                onChange={(newValue) => {
                  if (newValue) {
                    handleChange('timestamp', newValue.toISOString());
                  }
                }}
                slotProps={{
                  textField: {
                    fullWidth: true,
                    required: true,
                  },
                }}
              />
            </LocalizationProvider>

            <Typography variant="subtitle2" gutterBottom>
              Location
            </Typography>
            <Box display="flex" gap={2}>
              <TextField
                label="Latitude"
                type="number"
                value={formData.location?.lat || ''}
                onChange={(e) => handleChange('location', {
                  ...formData.location,
                  lat: parseFloat(e.target.value)
                })}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
              <TextField
                label="Longitude"
                type="number"
                value={formData.location?.lon || ''}
                onChange={(e) => handleChange('location', {
                  ...formData.location,
                  lon: parseFloat(e.target.value)
                })}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
            </Box>

            <TextField
              label="Speed (km/h)"
              type="number"
              value={formData.speed}
              onChange={(e) => handleChange('speed', parseFloat(e.target.value))}
              error={!!errors.speed}
              helperText={errors.speed}
              fullWidth
              required
              inputProps={{ min: 0, max: 300 }}
            />

            {vehicleType === 'ICE' ? (
              <TextField
                label="Fuel Level (%)"
                type="number"
                value={formData.fuel_level || ''}
                onChange={(e) => handleChange('fuel_level', parseFloat(e.target.value))}
                error={!!errors.fuel_level}
                helperText={errors.fuel_level}
                fullWidth
                required
                inputProps={{ min: 0, max: 100 }}
              />
            ) : (
              <TextField
                label="Battery Level (%)"
                type="number"
                value={formData.battery_level || ''}
                onChange={(e) => handleChange('battery_level', parseFloat(e.target.value))}
                error={!!errors.battery_level}
                helperText={errors.battery_level}
                fullWidth
                required
                inputProps={{ min: 0, max: 100 }}
              />
            )}

            <TextField
              label="Emissions (kg CO2)"
              type="number"
              value={formData.emissions}
              onChange={(e) => handleChange('emissions', parseFloat(e.target.value))}
              error={!!errors.emissions}
              helperText={errors.emissions}
              fullWidth
              required
              inputProps={{ min: 0, step: 0.1 }}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" disabled={loading}>
            {loading ? 'Saving...' : 'Save Telemetry'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default TelemetryForm; 