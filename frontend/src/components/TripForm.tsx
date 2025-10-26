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
} from '@mui/material';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { Trip } from '../types';

interface TripFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (trip: Omit<Trip, 'id'>) => Promise<void>;
  title: string;
}

const TripForm: React.FC<TripFormProps> = ({
  open,
  onClose,
  onSubmit,
  title,
}) => {
  const [formData, setFormData] = useState<Omit<Trip, 'id'>>({
    vehicle_id: '',
    driver_id: '',
    start_location: { lat: 40.7128, lon: -74.0060 },
    end_location: { lat: 40.7589, lon: -73.9851 },
    start_time: new Date().toISOString(),
    end_time: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(), // 2 hours later
    distance: 0,
    duration: 0,
    fuel_consumption: 0,
    battery_consumption: 0,
    cost: 0,
    purpose: 'business',
    status: 'planned',
    notes: '',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setFormData({
        vehicle_id: '',
        driver_id: '',
        start_location: { lat: 40.7128, lon: -74.0060 },
        end_location: { lat: 40.7589, lon: -73.9851 },
        start_time: new Date().toISOString(),
        end_time: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
        distance: 0,
        duration: 0,
        fuel_consumption: 0,
        battery_consumption: 0,
        cost: 0,
        purpose: 'business',
        status: 'planned',
        notes: '',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      });
      setErrors({});
    }
  }, [open]);

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.vehicle_id.trim()) {
      newErrors.vehicle_id = 'Vehicle ID is required';
    }
    if (!formData.driver_id.trim()) {
      newErrors.driver_id = 'Driver ID is required';
    }
    if (formData.distance < 0) {
      newErrors.distance = 'Distance must be non-negative';
    }
    if (formData.duration < 0) {
      newErrors.duration = 'Duration must be non-negative';
    }
    if (formData.cost < 0) {
      newErrors.cost = 'Cost must be non-negative';
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
      console.error('Error submitting trip:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field: keyof Omit<Trip, 'id'>, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const handleLocationChange = (type: 'start' | 'end', field: 'lat' | 'lon', value: number) => {
    const locationField = type === 'start' ? 'start_location' : 'end_location';
    setFormData(prev => ({
      ...prev,
      [locationField]: {
        ...prev[locationField],
        [field]: value
      }
    }));
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <Box display="flex" flexDirection="column" gap={2}>
            <Box display="flex" gap={2}>
              <TextField
                label="Vehicle ID"
                value={formData.vehicle_id}
                onChange={(e) => handleChange('vehicle_id', e.target.value)}
                error={!!errors.vehicle_id}
                helperText={errors.vehicle_id}
                fullWidth
                required
              />
              <TextField
                label="Driver ID"
                value={formData.driver_id}
                onChange={(e) => handleChange('driver_id', e.target.value)}
                error={!!errors.driver_id}
                helperText={errors.driver_id}
                fullWidth
                required
              />
            </Box>

            <FormControl fullWidth>
              <InputLabel>Purpose</InputLabel>
              <Select
                value={formData.purpose}
                onChange={(e) => handleChange('purpose', e.target.value)}
                label="Purpose"
              >
                <MenuItem value="business">Business</MenuItem>
                <MenuItem value="personal">Personal</MenuItem>
                <MenuItem value="delivery">Delivery</MenuItem>
              </Select>
            </FormControl>

            <FormControl fullWidth>
              <InputLabel>Status</InputLabel>
              <Select
                value={formData.status}
                onChange={(e) => handleChange('status', e.target.value)}
                label="Status"
              >
                <MenuItem value="planned">Planned</MenuItem>
                <MenuItem value="in_progress">In Progress</MenuItem>
                <MenuItem value="completed">Completed</MenuItem>
                <MenuItem value="cancelled">Cancelled</MenuItem>
              </Select>
            </FormControl>

            <LocalizationProvider dateAdapter={AdapterDateFns}>
              <Box display="flex" gap={2}>
                <DateTimePicker
                  label="Start Time"
                  value={new Date(formData.start_time)}
                  onChange={(newValue) => {
                    if (newValue) {
                      handleChange('start_time', newValue.toISOString());
                    }
                  }}
                  slotProps={{
                    textField: {
                      fullWidth: true,
                      required: true,
                    },
                  }}
                />
                <DateTimePicker
                  label="End Time"
                  value={new Date(formData.end_time)}
                  onChange={(newValue) => {
                    if (newValue) {
                      handleChange('end_time', newValue.toISOString());
                    }
                  }}
                  slotProps={{
                    textField: {
                      fullWidth: true,
                      required: true,
                    },
                  }}
                />
              </Box>
            </LocalizationProvider>

            <Typography variant="subtitle2" gutterBottom>
              Start Location
            </Typography>
            <Box display="flex" gap={2}>
              <TextField
                label="Latitude"
                type="number"
                value={formData.start_location.lat}
                onChange={(e) => handleLocationChange('start', 'lat', parseFloat(e.target.value))}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
              <TextField
                label="Longitude"
                type="number"
                value={formData.start_location.lon}
                onChange={(e) => handleLocationChange('start', 'lon', parseFloat(e.target.value))}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
            </Box>

            <Typography variant="subtitle2" gutterBottom>
              End Location
            </Typography>
            <Box display="flex" gap={2}>
              <TextField
                label="Latitude"
                type="number"
                value={formData.end_location.lat}
                onChange={(e) => handleLocationChange('end', 'lat', parseFloat(e.target.value))}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
              <TextField
                label="Longitude"
                type="number"
                value={formData.end_location.lon}
                onChange={(e) => handleLocationChange('end', 'lon', parseFloat(e.target.value))}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
            </Box>

            <Box display="flex" gap={2}>
              <TextField
                label="Distance (km)"
                type="number"
                value={formData.distance}
                onChange={(e) => handleChange('distance', parseFloat(e.target.value))}
                error={!!errors.distance}
                helperText={errors.distance}
                fullWidth
                inputProps={{ min: 0, step: 0.1 }}
              />
              <TextField
                label="Duration (hours)"
                type="number"
                value={formData.duration}
                onChange={(e) => handleChange('duration', parseFloat(e.target.value))}
                error={!!errors.duration}
                helperText={errors.duration}
                fullWidth
                inputProps={{ min: 0, step: 0.1 }}
              />
            </Box>

            <Box display="flex" gap={2}>
              <TextField
                label="Fuel Consumption (L)"
                type="number"
                value={formData.fuel_consumption}
                onChange={(e) => handleChange('fuel_consumption', parseFloat(e.target.value))}
                fullWidth
                inputProps={{ min: 0, step: 0.1 }}
              />
              <TextField
                label="Battery Consumption (kWh)"
                type="number"
                value={formData.battery_consumption}
                onChange={(e) => handleChange('battery_consumption', parseFloat(e.target.value))}
                fullWidth
                inputProps={{ min: 0, step: 0.1 }}
              />
            </Box>

            <TextField
              label="Cost ($)"
              type="number"
              value={formData.cost}
              onChange={(e) => handleChange('cost', parseFloat(e.target.value))}
              error={!!errors.cost}
              helperText={errors.cost}
              fullWidth
              inputProps={{ min: 0, step: 0.01 }}
            />

            <TextField
              label="Notes"
              value={formData.notes}
              onChange={(e) => handleChange('notes', e.target.value)}
              fullWidth
              multiline
              rows={3}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" disabled={loading}>
            {loading ? 'Saving...' : 'Save Trip'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default TripForm; 