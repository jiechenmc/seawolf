import { BrowserRouter as Router, Route, Routes } from 'react-router-dom'
import Login from './components/Login'
import Home from './components/Home'
import Exchange from './components/Exchange'
import Mining from './components/Mining'
import Proxy from './components/Proxy'
import Wallet from './components/Wallet'
import Settings from './components/Settings'
import { AppProvider } from './AppContext'

function App(): JSX.Element {
  return (
    <AppProvider>
      <Router>
        <Routes>
          <Route path="/" element={<Login />} />
          <Route path="/home" element={<Home />} />
          <Route path="/exchange" element={<Exchange />} />
          <Route path="/mining" element={<Mining />} />
          <Route path="/proxy" element={<Proxy />} />
          <Route path="/wallet" element={<Wallet />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Router>
    </AppProvider>
  )
}

export default App
