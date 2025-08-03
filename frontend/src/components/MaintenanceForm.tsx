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
} from '@mui/material';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { Maintenance } from '../types';

interface MaintenanceFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (maintenance: Omit<Maintenance, 'id'>) => Promise<void>;
  title: string;
}

const MaintenanceForm: React.FC<MaintenanceFormProps> = ({
  open,
  onClose,
  onSubmit,
  title,
}) => {
  const [formData, setFormData] = useState<Omit<Maintenance, 'id'>>({
    vehicle_id: '',
    service_type: 'oil_change',
    description: '',
    service_date: new Date().toISOString(),
    next_service_date: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days later
    mileage: 0,
    cost: 0,
    labor_cost: 0,
    parts_cost: 0,
    technician: '',
    service_location: '',
    status: 'scheduled',
    priority: 'medium',
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
        service_type: 'oil_change',
        description: '',
        service_date: new Date().toISOString(),
        next_service_date: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
        mileage: 0,
        cost: 0,
        labor_cost: 0,
        parts_cost: 0,
        technician: '',
        service_location: '',
        status: 'scheduled',
        priority: 'medium',
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
    if (!formData.description.trim()) {
      newErrors.description = 'Description is required';
    }
    if (formData.mileage < 0) {
      newErrors.mileage = 'Mileage must be non-negative';
    }
    if (formData.cost < 0) {
      newErrors.cost = 'Cost must be non-negative';
    }
    if (formData.labor_cost < 0) {
      newErrors.labor_cost = 'Labor cost must be non-negative';
    }
    if (formData.parts_cost < 0) {
      newErrors.parts_cost = 'Parts cost must be non-negative';
    }
    if (!formData.technician.trim()) {
      newErrors.technician = 'Technician is required';
    }
    if (!formData.service_location.trim()) {
      newErrors.service_location = 'Service location is required';
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
      console.error('Error submitting maintenance:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field: keyof Omit<Maintenance, 'id'>, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <Box display="flex" flexDirection="column" gap={2}>
            <TextField
              label="Vehicle ID"
              value={formData.vehicle_id}
              onChange={(e) => handleChange('vehicle_id', e.target.value)}
              error={!!errors.vehicle_id}
              helperText={errors.vehicle_id}
              fullWidth
              required
            />

            <FormControl fullWidth>
              <InputLabel>Service Type</InputLabel>
              <Select
                value={formData.service_type}
                onChange={(e) => handleChange('service_type', e.target.value)}
                label="Service Type"
              >
                <MenuItem value="oil_change">Oil Change</MenuItem>
                <MenuItem value="tire_rotation">Tire Rotation</MenuItem>
                <MenuItem value="brake_service">Brake Service</MenuItem>
                <MenuItem value="battery_service">Battery Service</MenuItem>
                <MenuItem value="inspection">Inspection</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => handleChange('description', e.target.value)}
              error={!!errors.description}
              helperText={errors.description}
              fullWidth
              multiline
              rows={2}
              required
            />

            <LocalizationProvider dateAdapter={AdapterDateFns}>
              <Box display="flex" gap={2}>
                <DatePicker
                  label="Service Date"
                  value={new Date(formData.service_date)}
                  onChange={(newValue) => {
                    if (newValue) {
                      handleChange('service_date', newValue.toISOString());
                    }
                  }}
                  slotProps={{
                    textField: {
                      fullWidth: true,
                      required: true,
                    },
                  }}
                />
                <DatePicker
                  label="Next Service Date"
                  value={new Date(formData.next_service_date)}
                  onChange={(newValue) => {
                    if (newValue) {
                      handleChange('next_service_date', newValue.toISOString());
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

            <Box display="flex" gap={2}>
              <TextField
                label="Mileage (km)"
                type="number"
                value={formData.mileage}
                onChange={(e) => handleChange('mileage', parseFloat(e.target.value))}
                error={!!errors.mileage}
                helperText={errors.mileage}
                fullWidth
                inputProps={{ min: 0 }}
              />
              <TextField
                label="Total Cost ($)"
                type="number"
                value={formData.cost}
                onChange={(e) => handleChange('cost', parseFloat(e.target.value))}
                error={!!errors.cost}
                helperText={errors.cost}
                fullWidth
                inputProps={{ min: 0, step: 0.01 }}
              />
            </Box>

            <Box display="flex" gap={2}>
              <TextField
                label="Labor Cost ($)"
                type="number"
                value={formData.labor_cost}
                onChange={(e) => handleChange('labor_cost', parseFloat(e.target.value))}
                error={!!errors.labor_cost}
                helperText={errors.labor_cost}
                fullWidth
                inputProps={{ min: 0, step: 0.01 }}
              />
              <TextField
                label="Parts Cost ($)"
                type="number"
                value={formData.parts_cost}
                onChange={(e) => handleChange('parts_cost', parseFloat(e.target.value))}
                error={!!errors.parts_cost}
                helperText={errors.parts_cost}
                fullWidth
                inputProps={{ min: 0, step: 0.01 }}
              />
            </Box>

            <TextField
              label="Technician"
              value={formData.technician}
              onChange={(e) => handleChange('technician', e.target.value)}
              error={!!errors.technician}
              helperText={errors.technician}
              fullWidth
              required
            />

            <TextField
              label="Service Location"
              value={formData.service_location}
              onChange={(e) => handleChange('service_location', e.target.value)}
              error={!!errors.service_location}
              helperText={errors.service_location}
              fullWidth
              required
            />

            <Box display="flex" gap={2}>
              <FormControl fullWidth>
                <InputLabel>Status</InputLabel>
                <Select
                  value={formData.status}
                  onChange={(e) => handleChange('status', e.target.value)}
                  label="Status"
                >
                  <MenuItem value="scheduled">Scheduled</MenuItem>
                  <MenuItem value="in_progress">In Progress</MenuItem>
                  <MenuItem value="completed">Completed</MenuItem>
                  <MenuItem value="cancelled">Cancelled</MenuItem>
                </Select>
              </FormControl>

              <FormControl fullWidth>
                <InputLabel>Priority</InputLabel>
                <Select
                  value={formData.priority}
                  onChange={(e) => handleChange('priority', e.target.value)}
                  label="Priority"
                >
                  <MenuItem value="low">Low</MenuItem>
                  <MenuItem value="medium">Medium</MenuItem>
                  <MenuItem value="high">High</MenuItem>
                  <MenuItem value="critical">Critical</MenuItem>
                </Select>
              </FormControl>
            </Box>

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
            {loading ? 'Saving...' : 'Save Maintenance'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default MaintenanceForm; 