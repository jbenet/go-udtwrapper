#ifndef WIN32
   #include <cstdlib>
   #include <netdb.h>
#else
   #include <winsock2.h>
   #include <ws2tcpip.h>
#endif
#include <fstream>
#include <iostream>
#include <cstring>
#include <udt.h>

using namespace std;

#ifndef WIN32
void* sendfile(void*);
#else
DWORD WINAPI sendfile(LPVOID);
#endif

int main(int argc, char* argv[])
{
   //usage: sendfile [server_port]
   if ((2 < argc) || ((2 == argc) && (0 == atoi(argv[1]))))
   {
      cout << "usage: sendfile [server_port]" << endl;
      return 0;
   }

   // use this function to initialize the UDT library
   UDT::udt_startup();

   addrinfo hints;
   addrinfo* res;

   memset(&hints, 0, sizeof(struct addrinfo));
   hints.ai_flags = AI_PASSIVE;
   hints.ai_family = AF_INET;
   hints.ai_socktype = SOCK_STREAM;

   string service("9000");
   if (2 == argc)
      service = argv[1];

   if (0 != getaddrinfo(NULL, service.c_str(), &hints, &res))
   {
      cout << "illegal port number or port is busy.\n" << endl;
      return 0;
   }

   UDTSOCKET serv = UDT::udt_socket(res->ai_family, res->ai_socktype, res->ai_protocol);

   // Windows UDP issue
   // For better performance, modify HKLM\System\CurrentControlSet\Services\Afd\Parameters\FastSendDatagramThreshold
#ifdef WIN32
   int mss = 1052;
   UDT::udt_setsockopt(serv, 0, UDT_MSS, &mss, sizeof(int));
#endif

   if (UDT::ERROR == UDT::udt_bind(serv, res->ai_addr, res->ai_addrlen))
   {
      cout << "bind: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return 0;
   }

   freeaddrinfo(res);

   cout << "server is ready at port: " << service << endl;

   UDT::udt_listen(serv, 10);

   sockaddr_storage clientaddr;
   int addrlen = sizeof(clientaddr);

   UDTSOCKET fhandle;

   while (true)
   {
      if (UDT::INVALID_SOCK == (fhandle = UDT::udt_accept(serv, (sockaddr*)&clientaddr, &addrlen)))
      {
         cout << "accept: " << UDT::udt_getlasterror().getErrorMessage() << endl;
         return 0;
      }

      char clienthost[NI_MAXHOST];
      char clientservice[NI_MAXSERV];
      getnameinfo((sockaddr *)&clientaddr, addrlen, clienthost, sizeof(clienthost), clientservice, sizeof(clientservice), NI_NUMERICHOST|NI_NUMERICSERV);
      cout << "new connection: " << clienthost << ":" << clientservice << endl;

      #ifndef WIN32
         pthread_t filethread;
         pthread_create(&filethread, NULL, sendfile, new UDTSOCKET(fhandle));
         pthread_detach(filethread);
      #else
         CreateThread(NULL, 0, sendfile, new UDTSOCKET(fhandle), 0, NULL);
      #endif
   }

   UDT::udt_close(serv);

   // use this function to release the UDT library
   UDT::udt_cleanup();

   return 0;
}

#ifndef WIN32
void* sendfile(void* usocket)
#else
DWORD WINAPI sendfile(LPVOID usocket)
#endif
{
   UDTSOCKET fhandle = *(UDTSOCKET*)usocket;
   delete (UDTSOCKET*)usocket;

   // aquiring file name information from client
   char file[1024];
   int len;

   if (UDT::ERROR == UDT::udt_recv(fhandle, (char*)&len, sizeof(int), 0))
   {
      cout << "recv: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return 0;
   }

   if (UDT::ERROR == UDT::udt_recv(fhandle, file, len, 0))
   {
      cout << "recv: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return 0;
   }
   file[len] = '\0';

   // open the file
   fstream ifs(file, ios::in | ios::binary);

   ifs.seekg(0, ios::end);
   int64_t size = ifs.tellg();
   ifs.seekg(0, ios::beg);

   // send file size information
   if (UDT::ERROR == UDT::udt_send(fhandle, (char*)&size, sizeof(int64_t), 0))
   {
      cout << "send: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return 0;
   }

   UDT::TRACEINFO trace;
   UDT::udt_perfmon(fhandle, &trace);

   // send the file
   int64_t offset = 0;
   if (UDT::ERROR == UDT::udt_sendfile(fhandle, ifs, offset, size))
   {
      cout << "sendfile: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return 0;
   }

   UDT::udt_perfmon(fhandle, &trace);
   cout << "speed = " << trace.mbpsSendRate << "Mbits/sec" << endl;

   UDT::udt_close(fhandle);

   ifs.close();

   #ifndef WIN32
      return NULL;
   #else
      return 0;
   #endif
}
