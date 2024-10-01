import { BrowserRouter as Router, Route, Routes } from 'react-router-dom'
import Login from './components/Login'
import Home from './components/Home'
import Exchange from './components/Exchange'
import Mining from './components/Mining'
import Wallet from './components/Wallet'
import Settings from './components/Settings'

function App(): JSX.Element {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Login />} />
        <Route path="/home" element={<Home />} />
        <Route path="/exchange" element={<Exchange />} />
        <Route path="/mining" element={<Mining />} />
        <Route path="/wallet" element={<Wallet />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Router>
  )
}

export default App
