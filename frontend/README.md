# Fleet Sustainability Dashboard - Frontend

A modern React TypeScript frontend for the Fleet Sustainability Dashboard that provides real-time monitoring and analytics for vehicle fleets.

## Features

- **Real-time Fleet Monitoring**: View all vehicles on an interactive map
- **Fleet Metrics**: Comprehensive analytics including emissions, EV percentage, and efficiency
- **Vehicle Details**: Detailed information for each vehicle with expandable rows
- **Time Range Filtering**: Filter data by custom date ranges or quick presets
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Modern UI**: Built with Material-UI for a professional look and feel

## Tech Stack

- **React 18** with TypeScript
- **Material-UI (MUI)** for UI components
- **Recharts** for data visualization
- **Axios** for API communication
- **Date-fns** for date manipulation

## Getting Started

### Prerequisites

- Node.js 16+ and npm
- Go backend running on `http://localhost:8080`

### Installation

1. Install dependencies:
```bash
npm install
```

2. Start the development server:
```bash
npm start
```

3. Open [http://localhost:3000](http://localhost:3000) in your browser

### Environment Variables

Create a `.env` file in the frontend directory:

```env
REACT_APP_API_URL=http://localhost:8080
```

## Project Structure

```
src/
├── components/          # React components
│   ├── Dashboard.tsx   # Main dashboard component
│   ├── FleetMap.tsx    # Fleet map visualization
│   ├── MetricsPanel.tsx # Metrics and charts
│   ├── VehicleList.tsx # Vehicle table
│   └── TimeRangeSelector.tsx # Date/time filter
├── services/           # API services
│   └── api.ts         # API client
├── types/              # TypeScript interfaces
│   └── index.ts       # Data models
└── App.tsx            # Main app component
```

## API Integration

The frontend connects to the Go backend API endpoints:

- `GET /api/telemetry` - Get vehicle telemetry data
- `GET /api/telemetry/metrics` - Get fleet metrics
- `GET /api/vehicles` - Get vehicle list

## Development

### Available Scripts

- `npm start` - Start development server
- `npm build` - Build for production
- `npm test` - Run tests
- `npm eject` - Eject from Create React App

### Code Style

- TypeScript for type safety
- Material-UI for consistent styling
- Functional components with hooks
- Proper error handling and loading states

## Deployment

1. Build the application:
```bash
npm run build
```

2. Serve the `build` directory with a static file server

## Contributing

1. Follow the existing code style
2. Add TypeScript types for new features
3. Include proper error handling
4. Test on different screen sizes

## License

This project is part of the Fleet Sustainability Dashboard.
