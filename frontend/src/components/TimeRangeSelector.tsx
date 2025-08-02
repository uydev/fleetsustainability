import React, { useState } from 'react';
import {
  Box,
  Button,
  Typography,
  Paper,
  Grid,
  Chip,
} from '@mui/material';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { format, subDays, startOfDay, endOfDay } from 'date-fns';

interface TimeRangeSelectorProps {
  onTimeRangeChange: (timeRange: { from?: string; to?: string }) => void;
}

const TimeRangeSelector: React.FC<TimeRangeSelectorProps> = ({ onTimeRangeChange }) => {
  const [fromDate, setFromDate] = useState<Date | null>(null);
  const [toDate, setToDate] = useState<Date | null>(null);

  const handleQuickSelect = (days: number) => {
    const to = new Date();
    const from = subDays(to, days);
    setFromDate(from);
    setToDate(to);
    
    onTimeRangeChange({
      from: format(startOfDay(from), "yyyy-MM-dd'T'HH:mm:ss"),
      to: format(endOfDay(to), "yyyy-MM-dd'T'HH:mm:ss"),
    });
  };

  const handleCustomRange = () => {
    if (fromDate && toDate) {
      onTimeRangeChange({
        from: format(startOfDay(fromDate), "yyyy-MM-dd'T'HH:mm:ss"),
        to: format(endOfDay(toDate), "yyyy-MM-dd'T'HH:mm:ss"),
      });
    }
  };

  const handleClear = () => {
    setFromDate(null);
    setToDate(null);
    onTimeRangeChange({});
  };

  const quickRanges = [
    { label: 'Last Hour', days: 0.04 },
    { label: 'Last 24h', days: 1 },
    { label: 'Last 7d', days: 7 },
    { label: 'Last 30d', days: 30 },
  ];

  return (
    <Paper sx={{ p: 2, mb: 3 }}>
      <Typography variant="h6" gutterBottom>
        Time Range Filter
      </Typography>
      
      <Grid container spacing={2} alignItems="center">
        {/* Quick Select Buttons */}
        <Grid item xs={12} md={6}>
          <Box display="flex" gap={1} flexWrap="wrap">
            {quickRanges.map((range) => (
              <Chip
                key={range.label}
                label={range.label}
                onClick={() => handleQuickSelect(range.days)}
                variant="outlined"
                clickable
              />
            ))}
            <Chip
              label="Clear"
              onClick={handleClear}
              variant="outlined"
              color="error"
              clickable
            />
          </Box>
        </Grid>

        {/* Custom Date Range */}
        <Grid item xs={12} md={6}>
          <LocalizationProvider dateAdapter={AdapterDateFns}>
            <Box display="flex" gap={2} alignItems="center">
              <DatePicker
                label="From Date"
                value={fromDate}
                onChange={(newValue) => setFromDate(newValue)}
                slotProps={{
                  textField: {
                    size: 'small',
                    fullWidth: true,
                  },
                }}
              />
              <DatePicker
                label="To Date"
                value={toDate}
                onChange={(newValue) => setToDate(newValue)}
                slotProps={{
                  textField: {
                    size: 'small',
                    fullWidth: true,
                  },
                }}
              />
              <Button
                variant="contained"
                size="small"
                onClick={handleCustomRange}
                disabled={!fromDate || !toDate}
              >
                Apply
              </Button>
            </Box>
          </LocalizationProvider>
        </Grid>
      </Grid>
    </Paper>
  );
};

export default TimeRangeSelector; 