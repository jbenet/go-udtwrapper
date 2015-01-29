#ifndef WIN32
   #include <arpa/inet.h>
   #include <netdb.h>
#else
   #include <winsock2.h>
   #include <ws2tcpip.h>
#endif
#include <fstream>
#include <iostream>
#include <cstdlib>
#include <cstring>
#include <udt.h>

using namespace std;

int main(int argc, char* argv[])
{
   if ((argc != 5) || (0 == atoi(argv[2])))
   {
      cout << "usage: recvfile server_ip server_port remote_filename local_filename" << endl;
      return -1;
   }

   // use this function to initialize the UDT library
   UDT::udt_startup();

   struct addrinfo hints, *peer;

   memset(&hints, 0, sizeof(struct addrinfo));
   hints.ai_flags = AI_PASSIVE;
   hints.ai_family = AF_INET;
   hints.ai_socktype = SOCK_STREAM;

   UDTSOCKET fhandle = UDT::udt_socket(hints.ai_family, hints.ai_socktype, hints.ai_protocol);

   if (0 != getaddrinfo(argv[1], argv[2], &hints, &peer))
   {
      cout << "incorrect server/peer address. " << argv[1] << ":" << argv[2] << endl;
      return -1;
   }

   // connect to the server, implict bind
   if (UDT::ERROR == UDT::udt_connect(fhandle, peer->ai_addr, peer->ai_addrlen))
   {
      cout << "connect: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return -1;
   }

   freeaddrinfo(peer);


   // send name information of the requested file
   int len = strlen(argv[3]);

   if (UDT::ERROR == UDT::udt_send(fhandle, (char*)&len, sizeof(int), 0))
   {
      cout << "send: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return -1;
   }

   if (UDT::ERROR == UDT::udt_send(fhandle, argv[3], len, 0))
   {
      cout << "send: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return -1;
   }

   // get size information
   int64_t size;

   if (UDT::ERROR == UDT::udt_recv(fhandle, (char*)&size, sizeof(int64_t), 0))
   {
      cout << "send: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return -1;
   }

   if (size < 0)
   {
      cout << "no such file " << argv[3] << " on the server\n";
      return -1;
   }

   // receive the file
   fstream ofs(argv[4], ios::out | ios::binary | ios::trunc);
   int64_t recvsize; 
   int64_t offset = 0;

   if (UDT::ERROR == (recvsize = UDT::udt_recvfile(fhandle, ofs, offset, size)))
   {
      cout << "recvfile: " << UDT::udt_getlasterror().getErrorMessage() << endl;
      return -1;
   }

   UDT::udt_close(fhandle);

   ofs.close();

   // use this function to release the UDT library
   UDT::udt_cleanup();

   return 0;
}
