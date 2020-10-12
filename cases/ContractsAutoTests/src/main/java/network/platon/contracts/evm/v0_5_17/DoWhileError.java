package network.platon.contracts.evm.v0_5_17;

import com.alaya.abi.solidity.TypeReference;
import com.alaya.abi.solidity.datatypes.Bool;
import com.alaya.abi.solidity.datatypes.Function;
import com.alaya.abi.solidity.datatypes.Type;
import com.alaya.crypto.Credentials;
import com.alaya.protocol.Web3j;
import com.alaya.protocol.core.RemoteCall;
import com.alaya.tx.Contract;
import com.alaya.tx.TransactionManager;
import com.alaya.tx.gas.GasProvider;
import java.util.Arrays;

/**
 * <p>Auto generated code.
 * <p><strong>Do not modify!</strong>
 * <p>Please use the <a href="https://github.com/PlatONnetwork/client-sdk-java/releases">platon-web3j command line tools</a>,
 * or the com.alaya.codegen.SolidityFunctionWrapperGenerator in the 
 * <a href="https://github.com/PlatONnetwork/client-sdk-java/tree/master/codegen">codegen module</a> to update.
 *
 * <p>Generated with web3j version 0.13.2.1.
 */
public class DoWhileError extends Contract {
    private static final String BINARY = "608060405234801561001057600080fd5b5060c48061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063a8a5cadc14602d575b600080fd5b6033604d565b604051808215151515815260200191505060405180910390f35b60008060566070565b90508060000160009054906101000a900460ff1691505090565b60005b600090508060000160009054906101000a900460ff166073579056fea265627a7a7231582037beb12de968d48a2f14818fc831de38e81be8da556b35ede63100c016c8c9eb64736f6c63430005110032";

    public static final String FUNC_GETDOWHILECONTROLRES = "getDoWhileControlRes";

    protected DoWhileError(String contractAddress, Web3j web3j, Credentials credentials, GasProvider contractGasProvider, Long chainId) {
        super(BINARY, contractAddress, web3j, credentials, contractGasProvider, chainId);
    }

    protected DoWhileError(String contractAddress, Web3j web3j, TransactionManager transactionManager, GasProvider contractGasProvider, Long chainId) {
        super(BINARY, contractAddress, web3j, transactionManager, contractGasProvider, chainId);
    }

    public RemoteCall<Boolean> getDoWhileControlRes() {
        final Function function = new Function(FUNC_GETDOWHILECONTROLRES, 
                Arrays.<Type>asList(), 
                Arrays.<TypeReference<?>>asList(new TypeReference<Bool>() {}));
        return executeRemoteCallSingleValueReturn(function, Boolean.class);
    }

    public static RemoteCall<DoWhileError> deploy(Web3j web3j, Credentials credentials, GasProvider contractGasProvider, Long chainId) {
        return deployRemoteCall(DoWhileError.class, web3j, credentials, contractGasProvider, BINARY,  "", chainId);
    }

    public static RemoteCall<DoWhileError> deploy(Web3j web3j, TransactionManager transactionManager, GasProvider contractGasProvider, Long chainId) {
        return deployRemoteCall(DoWhileError.class, web3j, transactionManager, contractGasProvider, BINARY,  "", chainId);
    }

    public static DoWhileError load(String contractAddress, Web3j web3j, Credentials credentials, GasProvider contractGasProvider, Long chainId) {
        return new DoWhileError(contractAddress, web3j, credentials, contractGasProvider, chainId);
    }

    public static DoWhileError load(String contractAddress, Web3j web3j, TransactionManager transactionManager, GasProvider contractGasProvider, Long chainId) {
        return new DoWhileError(contractAddress, web3j, transactionManager, contractGasProvider, chainId);
    }
}
