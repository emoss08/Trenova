export default function LoadingSkeleton() {
  return (
    <div className="flex flex-row min-h-screen justify-center items-center">
      <div className="border">
        <div className="flex flex-col sm:flex-row sm:justify-center sm:items-center">
          <div className="p-8">
            <p className="font-semibold text-lg mb-2">
              Monta is loading. Please wait.
            </p>
            <p className="text-sm text-gray-400 mt-1">
              If the operation exceeds a duration of 10 seconds, kindly verify
              the status of your internet connectivity. <br />
              In case of persistent difficulty, please get in touch with your
              designated system administrator.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
