import SideNav from './SideNav'
import NavBar from './NavBar'
import { FaRegClipboard } from 'react-icons/fa'


function Account(): JSX.Element {
  const handleCopyToClipboard = () => {
    navigator.clipboard
      .writeText('hello')
      .then(() => {
        console.log('copied to clipboard');
      })
      .catch((err) => {
        console.error('failed to copy due to: ', err);
      });
  };

  return (
    <div className="flex ml-52">
      <SideNav />
      {/* <NavBar /> */}
      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4"> Account</h1>
        <div className="border-b border-gray-300 mb-6"></div>
        {/* Wallet*/}
        <h2 className="text-xl font-semibold mb-2">Wallet</h2>
        <div className="border-b border-gray-300 mb-4"></div>
        <div className="flex justify-between mb-16">
          <div className="bg-white p-4 rounded-lg shadow-md w-1/2">
            <div className="flex items-center mb-3">
              <h3 className="text-lg font-semibold">Wallet ID: testingtesting</h3>
              <button
                onClick={handleCopyToClipboard}
                className="ml-2 p-1 hover:bg-gray-200 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Copy value to clipboard"
              >
                <FaRegClipboard className="text-xl" />
              </button>
            </div>
            <div className="flex-1 text-left">
              <h3 className="font-bold">Current Balance</h3>
              <p className="text-lg">0 BTC</p>
            </div>
          </div>
        </div>
        <h3 className="text-lg font-semibold mb-2">History</h3>
        <div className="overflow-x-auto border border-gray-300 rounded-lg mb-8">
          <div className="flex items-center p-2 border-b border-gray-300">
            <span className="flex-1 font-semibold">Date</span>
            <span className="flex-1 font-semibold">Transaction ID</span>
            <span className="flex-1 font-semibold">File</span>
            <span className="flex-1 font-semibold">Amount</span>
          </div>
        </div>

        {/*transfer*/}
        <h2 className="text-xl font-semibold mb-2">Transfer Settings</h2>
        <div className="border-b border-gray-300 mb-4 w-1/6"></div>

        <div className="mb-4">
          <label htmlFor="transfer-name" className="block text-sm font-medium text-gray-700 mb-1">
            Save Folder
          </label>
          <input
            type="text"
            id="transfer-name"
            className="mt-1 block w-1/3 border border-gray-300 rounded-md p-2"
            placeholder="Enter new save directory"
          />
        </div>

        <div className="mb-4">
          <label htmlFor="upload-limit" className="block text-sm font-medium text-gray-700 mb-1">
            Upload Limit
          </label>
          <div className="flex border border-gray-300 rounded-md p-2 w-1/3 items-center">
            <input
              type="number"
              id="upload-limit"
              className="block border-none outline-none w-full pr-4"
              placeholder="0"
              min="0"
              step="any"
            />
            <span className="text-gray-500">Mbps</span>
          </div>
        </div>

        <div className="mb-4">
          <label htmlFor="upload-limit" className="block text-sm font-medium text-gray-700 mb-1">
            Upload Limit
          </label>
          <div className="flex border border-gray-300 rounded-md p-2 w-1/3 items-center">
            <input
              type="number"
              id="upload-limit"
              className="block border-none outline-none w-full pr-4"
              placeholder="0"
              min="0"
              step="any"
            />
            <span className="text-gray-500">Mbps</span>
          </div>
        </div>

        <button className="mt-4 px-4 py-2 bg-[#737fa3] text-white font-semibold rounded-md hover:bg-[#7c85a3]">
          Save Changes
        </button>
      </div>
    </div>
  )
}

export default Account
