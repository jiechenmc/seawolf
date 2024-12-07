import SideNav from './SideNav'
import Navbar from './NavBar'

function Mining(): JSX.Element {
  return (
    <div className="flex ml-52">
      {/* <Navbar /> */}
      <SideNav />

      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Mining</h1>
      </div>
    </div>
  )
}

export default Mining
